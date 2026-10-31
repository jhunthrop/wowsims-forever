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
