package retribution

import (
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

// Holy Strike rank spell ids; source spellconst/paladin.json (see
// sim/paladin/holy_strike.go's holyStrikeRanks).
const (
	holyStrikeRank3SpellID = 1866  // level 20
	holyStrikeRank8SpellID = 10333 // level 60
)

// holyStrikeTalentString builds a talent string that grants only Sacred
// Arbiter, by finding PaladinTalents' sacred_arbiter field number through
// reflection rather than hard-coding its position: FillTalentsProto
// (sim/core/character.go) walks each tree's talent strings against
// sequential proto field numbers, offset by TalentTreeSizes, so the
// field's number minus the Holy+Protection tree sizes minus one gives its
// zero-based index inside the Retribution tree's string.
func holyStrikeTalentString(t *testing.T) string {
	t.Helper()

	fd := (&proto.PaladinTalents{}).ProtoReflect().Descriptor().Fields().ByName("sacred_arbiter")
	if fd == nil {
		t.Fatal("PaladinTalents has no sacred_arbiter field -- talent proto layout changed")
	}

	retributionOffset := paladin.TalentTreeSizes[0] + paladin.TalentTreeSizes[1]
	retributionIdx := int(fd.Number()) - retributionOffset - 1
	if retributionIdx < 0 {
		t.Fatalf("sacred_arbiter field number %d is not inside the Retribution tree (offset %d)", fd.Number(), retributionOffset)
	}

	return "-" + "-" + strings.Repeat("0", retributionIdx) + "1"
}

// TestHolyStrikeRankChain regression-tests engine-fixes brief 1's item 3:
// Holy Strike is new in Forever and had no engine file before this change.
// This builds a Retribution paladin at level 60 and at level 20 and
// confirms the rank each level should have learned (rank 8, id 10333 at
// 60; rank 3, id 1866 at 20 -- see holyStrikeRanks) actually casts and
// lands.
func TestHolyStrikeRankChain(t *testing.T) {
	for _, tc := range []struct {
		name    string
		level   int32
		spellID int32
	}{
		{name: "Level60Rank8", level: 60, spellID: holyStrikeRank8SpellID},
		{name: "Level20Rank3", level: 20, spellID: holyStrikeRank3SpellID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rotation := core.APLRotationFromJsonString(`{
				"type": "TypeAPL",
				"priorityList": [
					{"action": {"castSpell": {"spellId": {"spellId": ` + strconv.Itoa(int(tc.spellID)) + `}}}}
				]
			}`)

			// A same-level (+3) target instead of the package's usual
			// level 63 dummy: at level 20 the weapon-skill gap against a
			// level 63 target makes Holy Strike miss, dodge or get
			// parried almost every time, which is a real, separate
			// low-level-vs-high-level combat mechanic, not something
			// this rank chain test is about.
			target := core.NewDefaultTarget()
			target.Level = tc.level + 3

			player := core.WithSpec(&proto.Player{
				Class:              proto.Class_ClassPaladin,
				Race:               proto.Race_RaceHuman,
				Level:              tc.level,
				Equipment:          &proto.EquipmentSpec{},
				Buffs:              core.FullBuffs.Player,
				TalentsString:      Phase45RetTalents,
				Rotation:           rotation,
				DistanceFromTarget: 5,
			}, PlayerOptionsSealofRighteousness)

			raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

			result := core.RunRaidSim(&proto.RaidSimRequest{
				Raid: raid,
				Encounter: &proto.Encounter{
					Duration: 20,
					Targets:  []*proto.Target{target},
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

			metrics := findActionMetrics(result, tc.spellID)
			if metrics == nil {
				t.Fatalf("Holy Strike (spell %d) never appears in the action metrics -- it did not cast", tc.spellID)
			}

			var casts, hits int32
			for _, target := range metrics.Targets {
				casts += target.Casts
				hits += target.Hits + target.Crits
			}
			if casts == 0 {
				t.Errorf("Holy Strike casts = 0, want > 0")
			}
			if hits == 0 {
				t.Errorf("Holy Strike hits+crits = 0, want > 0 (it cast but never landed)")
			}
		})
	}
}

// sealOfTheCrusaderSpellID is Seal of the Crusader's top rank at 60
// (spellconst/paladin.json id 20308, scaling to level 60 from its level-52
// learn level). Seal of the Crusader is cast directly by its own spell id
// in this fork, not through PaladinOptions.PrimarySeal/"Cast Primary
// Seal" (that path only covers Seal of Righteousness and Seal of
// Command; see sim/paladin/sotc.go).
const sealOfTheCrusaderSpellID = 20308

// judgementOfTheCrusaderSpellID is the debuff core.JudgementOfTheCrusaderAura
// applies to the target.
const judgementOfTheCrusaderSpellID = 20303

// TestSacredArbiterRefreshesJudgement regression-tests the Sacred Arbiter
// talent (talents/paladin.json, Retribution tree, spell id 1311087):
// "Increases the damage of your Holy Strike ability by 20% and causes it
// to refresh all Judgement effects on the target."
//
// A Retribution paladin with Sacred Arbiter casts Seal of the Crusader and
// then Judgement once before pull, landing Judgement of the Crusader (a
// 10 s debuff, core.JudgementOfTheCrusaderAura), then casts only Holy
// Strike for the rest of a 30 s encounter -- no further Judgement casts.
// Without Sacred Arbiter's refresh, the debuff can only ever be up for its
// one 10 s application (an uptime near 8.6 s: it lands ~1.4 s before
// pull). With the refresh, every ~10 s Holy Strike cast resets it to
// full, so its uptime should track almost the whole encounter.
func TestSacredArbiterRefreshesJudgement(t *testing.T) {
	rotation := core.APLRotationFromJsonString(`{
		"type": "TypeAPL",
		"prepullActions": [
			{"action": {"castSpell": {"spellId": {"spellId": ` + strconv.Itoa(sealOfTheCrusaderSpellID) + `}}}, "doAtValue": {"const": {"val": "-1.5s"}}},
			{"action": {"castSpell": {"spellId": {"spellId": 20271}}}, "doAtValue": {"const": {"val": "-1.4s"}}}
		],
		"priorityList": [
			{"action": {"castSpell": {"spellId": {"spellId": ` + strconv.Itoa(holyStrikeRank8SpellID) + `}}}}
		]
	}`)

	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      holyStrikeTalentString(t),
		Rotation:           rotation,
		DistanceFromTarget: 5,
	}, &proto.Player_RetributionPaladin{
		RetributionPaladin: &proto.RetributionPaladin{
			Options: &proto.PaladinOptions{},
		},
	})

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	const duration = 30
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: duration,
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

	strikeMetrics := findActionMetrics(result, holyStrikeRank8SpellID)
	if strikeMetrics == nil {
		t.Fatalf("Holy Strike never appears in the action metrics -- it did not cast")
	}
	var strikeCasts int32
	for _, target := range strikeMetrics.Targets {
		strikeCasts += target.Casts
	}
	if strikeCasts < 2 {
		t.Fatalf("Holy Strike casts = %d, want >= 2 (need repeated casts to test the refresh)", strikeCasts)
	}

	targetMetrics := result.EncounterMetrics.Targets[0]
	var uptime float64
	found := false
	for _, aura := range targetMetrics.Auras {
		if aura.Id.GetSpellId() == judgementOfTheCrusaderSpellID { // core.JudgementOfTheCrusaderAura
			uptime = aura.UptimeSecondsAvg
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Judgement of the Crusader never appears in the target's aura metrics -- it never landed")
	}

	// A single, unrefreshed application landing ~1.4s before pull has at
	// most ~8.6s of uptime inside the encounter. Sacred Arbiter refreshing
	// it on every Holy Strike should keep it up far longer than that.
	const unrefreshedUptimeCeiling = 9.0
	if uptime <= unrefreshedUptimeCeiling {
		t.Errorf("Judgement of the Crusader uptime = %.1fs, want > %.1fs (Sacred Arbiter should have refreshed it on each Holy Strike cast)", uptime, unrefreshedUptimeCeiling)
	}
}
