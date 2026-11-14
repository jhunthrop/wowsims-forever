package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// unsealedWrathSwapRotation holds Seal of Righteousness from before the
// pull and swaps to Seal of Command once, nine seconds in.
const unsealedWrathSwapRotation = `{
	"type": "TypeAPL",
	"prepullActions": [
		{"action": {"castSpell": {"spellId": {"spellId": 20293}}}, "doAtValue": {"const": {"val": "-1.5s"}}}
	],
	"priorityList": [
		{"action": {"condition": {"and": {"vals": [
			{"cmp": {"op": "OpGe", "lhs": {"currentTime": {}}, "rhs": {"const": {"val": "9s"}}}},
			{"not": {"val": {"auraIsActive": {"auraId": {"spellId": 20920}}}}}
		]}}, "castSpell": {"spellId": {"spellId": 20920}}}}
	]
}`

// sealOfRighteousnessProcCasts runs the swap rotation for 12 seconds and
// counts Seal of Righteousness's weapon procs.
func sealOfRighteousnessProcCasts(t *testing.T, talents map[string]int) int32 {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talentString(t, talents),
		Rotation:           core.APLRotationFromJsonString(unsealedWrathSwapRotation),
		DistanceFromTarget: 5,
	}, &proto.Player_RetributionPaladin{
		RetributionPaladin: &proto.RetributionPaladin{
			Options: &proto.PaladinOptions{PrimarySeal: proto.PaladinSeal_Righteousness},
		},
	})
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 12, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	metrics := findActionMetrics(result, sealOfRighteousnessProcRank8SpellID)
	if metrics == nil {
		return 0
	}
	var casts int32
	for _, target := range metrics.Targets {
		casts += target.Casts
	}
	return casts
}

// Unsealed Wrath (client 1.60.1.70707): swings under Seal of Righteousness
// store up to three copies of it; the first swing that lands under Seal of
// Command consumes them, and Seal of Righteousness activates on every
// swing, so each copy is one more of its procs.
func TestUnsealedWrathReleasesTheStoredSealOnASwap(t *testing.T) {
	talents := map[string]int{"seal_of_command": 1}
	without := sealOfRighteousnessProcCasts(t, talents)
	if without < 3 {
		t.Fatalf("only %d swings landed under Seal of Righteousness before the swap; the test needs three", without)
	}
	talents["unsealed_wrath"] = 1
	with := sealOfRighteousnessProcCasts(t, talents)
	if got := with - without; got != 3 {
		t.Errorf("Unsealed Wrath added %d Seal of Righteousness procs on the swap, want 3 (one per stored copy)", got)
	}
}

// unsealedWrathShortSwapRotation holds Seal of Righteousness for two unarmed
// swings (0 and 2 s) and swaps to Seal of Command right after the second
// (the rotation runs after a swing lands, so a 1.5 s floor is the 2 s
// swing), so the swing at 4 s releases what the two swings stored: two
// copies, unless an extra attack in the window stored a third.
const unsealedWrathShortSwapRotation = `{
	"type": "TypeAPL",
	"prepullActions": [
		{"action": {"castSpell": {"spellId": {"spellId": 20293}}}, "doAtValue": {"const": {"val": "-1.5s"}}}
	],
	"priorityList": [
		{"action": {"condition": {"and": {"vals": [
			{"cmp": {"op": "OpGe", "lhs": {"currentTime": {}}, "rhs": {"const": {"val": "1.5s"}}}},
			{"not": {"val": {"auraIsActive": {"auraId": {"spellId": 20920}}}}}
		]}}, "castSpell": {"spellId": {"spellId": 20920}}}}
	]
}`

// sealOfRighteousnessProcCastsUnderWindfury runs the short swap rotation
// for 8 seconds under the raid's Windfury Totem at a seed and counts Seal
// of Righteousness's weapon procs and the totem's landed strikes (10610,
// a main-hand strike since push 113887, core.ApplyWindfury).
func sealOfRighteousnessProcCastsUnderWindfury(t *testing.T, talents map[string]int, seed int64) (procs int32, windfuryStrikes int32) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talentString(t, talents),
		Rotation:           core.APLRotationFromJsonString(unsealedWrathShortSwapRotation),
		DistanceFromTarget: 5,
	}, &proto.Player_RetributionPaladin{
		RetributionPaladin: &proto.RetributionPaladin{
			Options: &proto.PaladinOptions{PrimarySeal: proto.PaladinSeal_Righteousness},
		},
	})
	raidBuffs := &proto.RaidBuffs{WindfuryTotem: true}
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, raidBuffs, core.FullBuffs.Debuffs)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 8, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: seed, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	if metrics := findActionMetrics(result, sealOfRighteousnessProcRank8SpellID); metrics != nil {
		for _, target := range metrics.Targets {
			procs += target.Casts
		}
	}
	for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
		if action.Id.GetSpellId() == core.WindfuryStrikeSpellId[3] {
			for _, target := range action.Targets {
				windfuryStrikes += target.Hits
			}
		}
	}
	return procs, windfuryStrikes
}

// Blizzard's 11 November 2026 hotfix: "Unsealed Fury no longer generates
// echoes from Windfury Totem attacks". Two unarmed swings land under Seal
// of Righteousness before the swap, so the swing under Seal of Command
// releases at most two copies whatever Windfury adds in the window; before
// the fix a Windfury extra attack in the window stored a third. Since push
// 113887 the totem's attack is a yellow strike, which no Seal swing rule
// counts, so the rule holds by construction; the test keeps it held.
func TestUnsealedWrathStoresNoCopyFromAWindfuryAttack(t *testing.T) {
	const seeds = 40
	const swingsBeforeTheSwap = 2
	var windfuryStrikesSeen int32
	for seed := int64(1); seed <= seeds; seed++ {
		talents := map[string]int{"seal_of_command": 1}
		without, _ := sealOfRighteousnessProcCastsUnderWindfury(t, talents, seed)
		talents["unsealed_wrath"] = 1
		with, windfuryStrikes := sealOfRighteousnessProcCastsUnderWindfury(t, talents, seed)
		windfuryStrikesSeen += windfuryStrikes
		if released := with - without; released > swingsBeforeTheSwap {
			t.Errorf("seed %d: Unsealed Wrath released %d copies, more than the %d swings that landed before the swap: a Windfury attack stored one", seed, released, swingsBeforeTheSwap)
		}
	}
	if windfuryStrikesSeen == 0 {
		t.Fatal("Windfury Totem never struck over the seeds; the test exercises nothing")
	}
}
