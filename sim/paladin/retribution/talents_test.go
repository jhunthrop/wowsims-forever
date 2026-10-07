package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// newTalentTestUnit builds a level-60 Retribution Paladin carrying only
// the given talent string and returns its Unit for direct Spell/stat
// inspection -- no rotation runs, so this is for structural assertions
// (a spell's registered DamageMultiplier, Cost, CastTime, CD, or a stat
// dependency's effect on GetStat), not for proc-rate or metrics
// assertions, which need an actual sim (see TestSanctifiedJudgement... /
// TestTwistOfLightEcho... below).
func newTalentTestUnit(t *testing.T, talents string) (*RetributionPaladin, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassPaladin,
			Race:          proto.Race_RaceHuman,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: talents,
		},
		&proto.Player_RetributionPaladin{
			RetributionPaladin: &proto.RetributionPaladin{
				Options: &proto.PaladinOptions{},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}

	env, _, _ := core.NewEnvironment(raid, encounter, true)
	agent := env.Raid.Parties[0].Players[0]
	ret, ok := agent.(*RetributionPaladin)
	if !ok {
		t.Fatalf("player agent is %T, want *RetributionPaladin", agent)
	}
	return ret, &ret.Paladin.Character.Unit
}

// Level 60 top-rank spell ids this file checks against (see sor.go,
// soc.go, consecration.go, holy_wrath.go, exorcism.go, hammer_of_wrath.go
// rank tables).
const (
	judgementOfRighteousnessRank8SpellID = 20286
	sealOfRighteousnessProcRank8SpellID  = 25713
	consecrationRank5SpellID             = 20924
	holyWrathRank2SpellID                = 10318
	exorcismRank6SpellID                 = 10314
	hammerOfWrathRank3SpellID            = 24239
)

func TestImprovedSealsIncreasesSealAndJudgementDamage(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"improved_seals": 3}))

	baseJudge := baseUnit.GetSpell(core.ActionID{SpellID: judgementOfRighteousnessRank8SpellID})
	talentedJudge := talentedUnit.GetSpell(core.ActionID{SpellID: judgementOfRighteousnessRank8SpellID})
	if baseJudge == nil || talentedJudge == nil {
		t.Fatalf("Judgement of Righteousness rank 8 not registered: base=%v talented=%v", baseJudge, talentedJudge)
	}
	wantMultiplier := baseJudge.DamageMultiplier * 1.15
	if talentedJudge.DamageMultiplier < wantMultiplier-1e-9 || talentedJudge.DamageMultiplier > wantMultiplier+1e-9 {
		t.Errorf("Judgement of Righteousness DamageMultiplier = %v, want %v (base %v * 1.15)", talentedJudge.DamageMultiplier, wantMultiplier, baseJudge.DamageMultiplier)
	}

	baseProc := baseUnit.GetSpell(core.ActionID{SpellID: sealOfRighteousnessProcRank8SpellID})
	talentedProc := talentedUnit.GetSpell(core.ActionID{SpellID: sealOfRighteousnessProcRank8SpellID})
	if baseProc == nil || talentedProc == nil {
		t.Fatalf("Seal of Righteousness proc rank 8 not registered: base=%v talented=%v", baseProc, talentedProc)
	}
	wantProcMultiplier := baseProc.DamageMultiplier * 1.15
	if talentedProc.DamageMultiplier < wantProcMultiplier-1e-9 || talentedProc.DamageMultiplier > wantProcMultiplier+1e-9 {
		t.Errorf("Seal of Righteousness proc DamageMultiplier = %v, want %v", talentedProc.DamageMultiplier, wantProcMultiplier)
	}
}

func TestReverenceIncreasesSpiritRegenWhileCasting(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"reverence": 3}))

	if baseUnit.PseudoStats.SpiritRegenRateCasting != 0 {
		t.Errorf("base SpiritRegenRateCasting = %v, want 0", baseUnit.PseudoStats.SpiritRegenRateCasting)
	}
	const want = 0.30
	got := talentedUnit.PseudoStats.SpiritRegenRateCasting
	if got < want-1e-9 || got > want+1e-9 {
		t.Errorf("rank 3 Reverence SpiritRegenRateCasting = %v, want %v", got, want)
	}
}

func TestPurifyingPowerReducesExorcismAndHolyWrathCooldown(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"purifying_power": 2}))

	for _, spellID := range []int32{exorcismRank6SpellID, holyWrathRank2SpellID} {
		base := baseUnit.GetSpell(core.ActionID{SpellID: spellID})
		talented := talentedUnit.GetSpell(core.ActionID{SpellID: spellID})
		if base == nil || talented == nil {
			t.Fatalf("spell %d not registered: base=%v talented=%v", spellID, base, talented)
		}
		wantCD := base.CD.Duration * 67 / 100 // rank 2: -33%
		if talented.CD.Duration != wantCD {
			t.Errorf("spell %d CD = %v, want %v (base %v - 33%%)", spellID, talented.CD.Duration, wantCD, base.CD.Duration)
		}
	}
}

func TestDivinePrecisionAddsHolySpellHit(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"divine_precision": 3}))

	base := baseUnit.GetSpell(core.ActionID{SpellID: exorcismRank6SpellID})
	talented := talentedUnit.GetSpell(core.ActionID{SpellID: exorcismRank6SpellID})
	if base == nil || talented == nil {
		t.Fatalf("Exorcism rank 6 not registered: base=%v talented=%v", base, talented)
	}
	wantBonus := 18.0 * core.HitRatingPerHitChance
	gotBonus := talented.BonusHitRating - base.BonusHitRating
	if gotBonus < wantBonus-1e-6 || gotBonus > wantBonus+1e-6 {
		t.Errorf("rank 3 Divine Precision BonusHitRating delta on Exorcism = %v, want %v (18%% hit)", gotBonus, wantBonus)
	}
}

func TestConsecratedGroundIncreasesConsecrationDamage(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"consecrated_ground": 2}))

	base := baseUnit.GetSpell(core.ActionID{SpellID: consecrationRank5SpellID})
	talented := talentedUnit.GetSpell(core.ActionID{SpellID: consecrationRank5SpellID})
	if base == nil || talented == nil {
		t.Fatalf("Consecration rank 5 not registered: base=%v talented=%v", base, talented)
	}
	wantMultiplier := base.DamageMultiplier * 1.10
	if talented.DamageMultiplier < wantMultiplier-1e-9 || talented.DamageMultiplier > wantMultiplier+1e-9 {
		t.Errorf("rank 2 Consecrated Ground Consecration DamageMultiplier = %v, want %v", talented.DamageMultiplier, wantMultiplier)
	}
}

func TestHolyConduitReducesRotationManaCosts(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"holy_conduit": 2}))

	for _, spellID := range []int32{consecrationRank5SpellID, holyWrathRank2SpellID, exorcismRank6SpellID, hammerOfWrathRank3SpellID} {
		base := baseUnit.GetSpell(core.ActionID{SpellID: spellID})
		talented := talentedUnit.GetSpell(core.ActionID{SpellID: spellID})
		if base == nil || talented == nil {
			t.Fatalf("spell %d not registered: base=%v talented=%v", spellID, base, talented)
		}
		baseCost := base.Cost.GetCurrentCost()
		talentedCost := talented.Cost.GetCurrentCost()
		wantCost := baseCost * 0.6 // rank 2: -40%
		if talentedCost < wantCost-1e-6 || talentedCost > wantCost+1e-6 {
			t.Errorf("spell %d mana cost = %v, want %v (base %v - 40%%)", spellID, talentedCost, wantCost, baseCost)
		}
	}
}

func TestInstrumentOfLawReducesHammerOfWrathCastTime(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, rank1Unit := newTalentTestUnit(t, talentString(t, map[string]int{"instrument_of_law": 1}))
	_, rank2Unit := newTalentTestUnit(t, talentString(t, map[string]int{"instrument_of_law": 2}))

	base := baseUnit.GetSpell(core.ActionID{SpellID: hammerOfWrathRank3SpellID})
	rank1 := rank1Unit.GetSpell(core.ActionID{SpellID: hammerOfWrathRank3SpellID})
	rank2 := rank2Unit.GetSpell(core.ActionID{SpellID: hammerOfWrathRank3SpellID})
	if base == nil || rank1 == nil || rank2 == nil {
		t.Fatalf("Hammer of Wrath rank 3 not registered: base=%v rank1=%v rank2=%v", base, rank1, rank2)
	}

	if want := base.DefaultCast.CastTime - 500_000_000; rank1.DefaultCast.CastTime != want {
		t.Errorf("rank 1 Instrument of Law Hammer of Wrath cast time = %v, want %v", rank1.DefaultCast.CastTime, want)
	}
	if rank2.DefaultCast.CastTime != 0 {
		t.Errorf("rank 2 Instrument of Law Hammer of Wrath cast time = %v, want 0 (base 1s - 1s)", rank2.DefaultCast.CastTime)
	}
}

// Champion of the Light is 20/40/60% of Intellect: the live text and
// Blizzard's 1 October 2026 notes; the earlier table said 33/66/100%.
func TestChampionOfTheLightAddsSpellPowerFromIntellect(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, "")
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"champion_of_the_light": 3}))

	baseIntellect := baseUnit.GetStat(stats.Intellect)
	if baseIntellect <= 0 {
		t.Fatalf("base Intellect = %v, want > 0 (test needs a nonzero base to observe the dependency)", baseIntellect)
	}

	wantSpellPower := baseUnit.GetStat(stats.SpellPower) + talentedUnit.GetStat(stats.Intellect)*0.60
	gotSpellPower := talentedUnit.GetStat(stats.SpellPower)
	if gotSpellPower < wantSpellPower-1e-6 || gotSpellPower > wantSpellPower+1e-6 {
		t.Errorf("rank 3 Champion of the Light SpellPower = %v, want %v (60%% of %v Intellect)", gotSpellPower, wantSpellPower, talentedUnit.GetStat(stats.Intellect))
	}
}

func TestSealOfCommandRequiresTalent(t *testing.T) {
	_, unit := newTalentTestUnit(t, "")

	if spell := unit.GetSpell(core.ActionID{SpellID: 20920}); spell != nil {
		t.Errorf("Seal of Command rank 5 (20920) registered without the seal_of_command talent")
	}
	if spell := unit.GetSpell(core.ActionID{SpellID: 20966}); spell != nil {
		t.Errorf("Judgement of Command rank 5 (20966) registered without the seal_of_command talent")
	}
}

func TestTwistOfLightReducesSealManaCost(t *testing.T) {
	_, baseUnit := newTalentTestUnit(t, talentString(t, map[string]int{"seal_of_command": 1}))
	_, talentedUnit := newTalentTestUnit(t, talentString(t, map[string]int{"seal_of_command": 1, "twist_of_light": 1}))

	for _, spellID := range []int32{20293 /* Seal of Righteousness rank 8 */, 20920 /* Seal of Command rank 5 */} {
		base := baseUnit.GetSpell(core.ActionID{SpellID: spellID})
		talented := talentedUnit.GetSpell(core.ActionID{SpellID: spellID})
		if base == nil || talented == nil {
			t.Fatalf("spell %d not registered: base=%v talented=%v", spellID, base, talented)
		}
		baseCost := base.Cost.GetCurrentCost()
		talentedCost := talented.Cost.GetCurrentCost()
		wantCost := baseCost * 0.8
		if talentedCost < wantCost-1e-6 || talentedCost > wantCost+1e-6 {
			t.Errorf("spell %d mana cost = %v, want %v (base %v - 20%%)", spellID, talentedCost, wantCost, baseCost)
		}
	}
}

// TestSanctifiedJudgementRefundsMana uses rank 3 (100% proc chance) for
// determinism: every Judgement cast must return mana, so a single cast
// is enough to prove the hook fires and AddMana is reached, without
// depending on sim.Proc's RNG draw.
func TestSanctifiedJudgementRefundsMana(t *testing.T) {
	rotation := core.APLRotationFromJsonString(`{
		"type": "TypeAPL",
		"prepullActions": [
			{"action": {"castSpell": {"spellId": {"spellId": 20293}}}, "doAtValue": {"const": {"val": "-1.5s"}}}
		],
		"priorityList": [
			{"action": {"castSpell": {"spellId": {"spellId": 20271}}}}
		]
	}`)

	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talentString(t, map[string]int{"sanctified_judgement": 3}),
		Rotation:           rotation,
		DistanceFromTarget: 5,
	}, PlayerOptionsSealofRighteousness)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 15,
			Targets:  []*proto.Target{core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{
			Iterations: 1,
			RandomSeed: 1,
			IsTest:     true,
		},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}

	judgeMetrics := findActionMetrics(result, 20271)
	if judgeMetrics == nil {
		t.Fatalf("Judgement never appears in the action metrics -- it did not cast")
	}
	var judgeCasts int32
	for _, target := range judgeMetrics.Targets {
		judgeCasts += target.Casts
	}
	if judgeCasts == 0 {
		t.Fatalf("Judgement casts = 0, want > 0")
	}

	player0 := result.RaidMetrics.Parties[0].Players[0]
	var gain float64
	found := false
	for _, resource := range player0.Resources {
		if resource.Id.GetSpellId() == 1311074 { // Sanctified Judgement (sim/paladin/talents.go's sanctifiedJudgementActionID)
			gain = resource.Gain
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Sanctified Judgement never appears in the resource metrics -- it never refunded mana")
	}
	if gain <= 0 {
		t.Errorf("Sanctified Judgement mana gain = %v, want > 0", gain)
	}
}

// TestTwistOfLightEchoAppliesReplacedSealOnNextMeleeHit: a paladin with
// Seal of Command and Twist of Light casts Seal of Righteousness, then
// immediately replaces it with Seal of Command before pull and never
// recasts either Seal again. Seal of Righteousness's own weapon proc
// (sealOfRighteousnessProcRank8SpellID) can therefore only land through
// Twist of Light's Echo -- the SoR aura that would otherwise drive it is
// long inactive -- so exactly one such cast (the Echo, consumed on the
// first landed white hit) is this test's signature that
// paladin.go's grantEchoOfSeal and talents.go's registerTwistOfLight
// fired.
func TestTwistOfLightEchoAppliesReplacedSealOnNextMeleeHit(t *testing.T) {
	rotation := core.APLRotationFromJsonString(`{
		"type": "TypeAPL",
		"prepullActions": [
			{"action": {"castSpell": {"spellId": {"spellId": 20293}}}, "doAtValue": {"const": {"val": "-3s"}}},
			{"action": {"castSpell": {"spellId": {"spellId": 20920}}}, "doAtValue": {"const": {"val": "-1.5s"}}}
		],
		"priorityList": []
	}`)

	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talentString(t, map[string]int{"seal_of_command": 1, "twist_of_light": 1}),
		Rotation:           rotation,
		DistanceFromTarget: 5,
	}, &proto.Player_RetributionPaladin{
		RetributionPaladin: &proto.RetributionPaladin{
			Options: &proto.PaladinOptions{PrimarySeal: proto.PaladinSeal_Command},
		},
	})

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 12,
			Targets:  []*proto.Target{core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{
			Iterations: 1,
			RandomSeed: 1,
			IsTest:     true,
		},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}

	echoMetrics := findActionMetrics(result, sealOfRighteousnessProcRank8SpellID)
	if echoMetrics == nil {
		t.Fatalf("Seal of Righteousness proc (spell %d) never appears in the action metrics -- the Echo did not fire", sealOfRighteousnessProcRank8SpellID)
	}
	var casts int32
	for _, target := range echoMetrics.Targets {
		casts += target.Casts
	}
	if casts != 1 {
		t.Errorf("Seal of Righteousness proc casts = %d, want exactly 1 (the single Echo of Light, since Seal of Command has been active since -1.5s and was never swapped again)", casts)
	}
}
