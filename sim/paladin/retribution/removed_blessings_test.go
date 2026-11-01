package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Forever removes Blessing of Light (19977-19979), Greater Blessing of Light
// (25890), Blessing of Salvation (1038) and Greater Blessing of Salvation
// (25895) from the paladin's spellbook: the 70707 hotfix push sets their
// SpellName rows to status 2 (removed), Wowhead Forever's tooltips for them
// return 404, and wowforevertalents' 31 October notes list them as "gone
// from the paladin spellbook". A level 60 paladin registers none of them.
func TestRemovedBlessingsAreNotRegistered(t *testing.T) {
	raid := core.SinglePlayerRaidProto(utilitySealPaladin(60, `{"type": "TypeAPL"}`), core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	unit := &sim.Raid.Parties[0].Players[0].GetCharacter().Unit
	for _, id := range []int32{19977, 19978, 19979, 25890, 1038, 25895} {
		if unit.GetSpell(core.ActionID{SpellID: id}) != nil {
			t.Errorf("spell %d is registered; Forever removed it", id)
		}
	}
}
