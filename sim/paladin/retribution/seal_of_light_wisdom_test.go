package retribution

import (
	"strconv"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

// Seal of Light and Seal of Wisdom from the 1.60.1.70009 client
// (data/builds/1.60.1.70009/raw; rows quoted in sim/paladin/utility_seals.go).

func utilitySealPaladin(level int32, rotation string) *proto.Player {
	return core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              level,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		Rotation:           core.APLRotationFromJsonString(rotation),
		DistanceFromTarget: 5,
	}, &proto.Player_RetributionPaladin{
		RetributionPaladin: &proto.RetributionPaladin{Options: &proto.PaladinOptions{}},
	})
}

func prepullCast(spellID int32, at string) string {
	return `{"action": {"castSpell": {"spellId": {"spellId": ` + strconv.Itoa(int(spellID)) + `}}}, "doAtValue": {"const": {"val": "` + at + `"}}}`
}

func runUtilitySeal(t *testing.T, level int32, duration float64, prepull ...string) *proto.RaidSimResult {
	t.Helper()
	return runUtilitySealUnder(t, core.FullBuffs.Debuffs, level, duration, prepull...)
}

func runUtilitySealUnder(t *testing.T, debuffs *proto.Debuffs, level int32, duration float64, prepull ...string) *proto.RaidSimResult {
	t.Helper()
	rotation := `{"type": "TypeAPL", "prepullActions": [`
	for i, action := range prepull {
		if i > 0 {
			rotation += ","
		}
		rotation += action
	}
	rotation += `], "priorityList": []}`
	raid := core.SinglePlayerRaidProto(utilitySealPaladin(level, rotation), core.FullBuffs.Party, core.FullBuffs.Raid, debuffs)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: duration, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	return result
}

func TestUtilitySealCastsAreRegisteredAtEveryLearnedRank(t *testing.T) {
	cases := []struct {
		level int32
		ids   []int32
		costs []float64
	}{
		{60, []int32{20165, 20347, 20348, 20349}, []float64{110, 140, 180, 210}}, // Seal of Light, levels 30/40/50/60
		{60, []int32{20166, 20356, 20357}, []float64{135, 170, 200}},             // Seal of Wisdom, levels 38/48/58
		{38, []int32{20165}, []float64{110}},                                     // a level 38 paladin has rank 1 of Light
		{38, []int32{20166}, []float64{135}},                                     // and rank 1 of Wisdom
		{29, nil, nil},                                                           // neither before level 30
	}
	for _, c := range cases {
		env, _, _ := core.NewEnvironment(core.SinglePlayerRaidProto(utilitySealPaladin(c.level, `{"type": "TypeAPL"}`), core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs), &proto.Encounter{Duration: 10, Targets: []*proto.Target{core.NewDefaultTarget()}}, true)
		unit := &env.Raid.Parties[0].Players[0].(*RetributionPaladin).Paladin.Character.Unit
		for i, id := range c.ids {
			spell := unit.GetSpell(core.ActionID{SpellID: id})
			if spell == nil {
				t.Errorf("level %d: seal spell %d is not registered", c.level, id)
				continue
			}
			if spell.DefaultCast.Cost != c.costs[i] {
				t.Errorf("level %d: %d costs %v, want %v", c.level, id, spell.DefaultCast.Cost, c.costs[i])
			}
		}
	}
	env, _, _ := core.NewEnvironment(core.SinglePlayerRaidProto(utilitySealPaladin(29, `{"type": "TypeAPL"}`), core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs), &proto.Encounter{Duration: 10, Targets: []*proto.Target{core.NewDefaultTarget()}}, true)
	for _, id := range []int32{20165, 20166} {
		if env.Raid.Parties[0].Players[0].GetCharacter().GetSpell(core.ActionID{SpellID: id}) != nil {
			t.Errorf("a level 29 paladin has seal spell %d", id)
		}
	}
}

// Seal of Light: "giving each melee attack a chance to heal the Paladin for
// 94" at rank 4 (spell 20340, effect 10, base points 94, no coefficient).
func TestSealOfLightHealsThePaladinOnMeleeHits(t *testing.T) {
	const duration = 300
	result := runUtilitySeal(t, 60, duration, prepullCast(20349, "-1.5s"))
	heal := findActionMetrics(result, 20340)
	if heal == nil {
		t.Fatal("Seal of Light never healed")
	}
	var triggers int32
	var healing float64
	for _, target := range heal.Targets {
		triggers += target.Casts
		healing += target.Healing
	}
	if triggers == 0 {
		t.Fatal("Seal of Light heal casts = 0")
	}
	if got := healing / float64(triggers); got != 94 {
		t.Errorf("each Seal of Light heal = %v, want 94", got)
	}
	// 15 procs a minute for the seal's 30 s.
	want := paladin.UtilitySealProcsPerMinute * 30 / 60
	if float64(triggers) < 0.4*want || float64(triggers) > 1.8*want {
		t.Errorf("Seal of Light healed %d times in its 30 s, want about %.0f", triggers, want)
	}
}

// Seal of Wisdom: "a chance to restore 90 of the Paladin's mana" at rank 3
// (spell 20351, effect 30, base points 90).
func TestSealOfWisdomRestoresManaOnMeleeHits(t *testing.T) {
	result := runUtilitySeal(t, 60, 300, prepullCast(20357, "-1.5s"))
	player := result.RaidMetrics.Parties[0].Players[0]
	var events int32
	var gain float64
	for _, resource := range player.Resources {
		if resource.Type == proto.ResourceType_ResourceTypeMana && resource.Id.GetSpellId() == 20351 {
			events += resource.Events
			gain += resource.Gain
		}
	}
	if events == 0 {
		t.Fatal("Seal of Wisdom never restored mana")
	}
	if got := gain / float64(events); got != 90 {
		t.Errorf("each Seal of Wisdom return = %v, want 90", got)
	}
}

// Judging a seal lays its judgement on the target for 40 seconds: Light and
// Wisdom are the raid debuffs the preset also names.
func TestJudgingASealAppliesItsJudgement(t *testing.T) {
	cases := []struct {
		name  string
		seal  int32
		aura  int32
		label string
	}{
		{"light", 20349, 20346, "Judgement of Light"},
		{"wisdom", 20357, 20355, "Judgement of Wisdom"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// The target carries no judgement of its own.
			bare := &proto.Debuffs{}
			result := runUtilitySealUnder(t, bare, 60, 60, prepullCast(c.seal, "-3s"), prepullCast(20271, "-1.5s"))
			found := false
			for _, aura := range result.EncounterMetrics.Targets[0].Auras {
				if aura.Id.GetSpellId() == c.aura {
					found = true
					if aura.UptimeSecondsAvg < 30 {
						t.Errorf("%s uptime = %.1fs, want about 40s less the lead-in", c.label, aura.UptimeSecondsAvg)
					}
				}
			}
			if !found {
				t.Errorf("%s never appears in the target's auras", c.label)
			}
		})
	}
}
