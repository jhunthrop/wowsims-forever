package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Round 24 (hunter lane): the rank sweep at the first two levels after each
// new rank found Mongoose Bite and Lacerate registering only their highest
// learned rank, so a forced lower rank (spec-breakdown -rank, or a rank-choice
// row) found no spell and the line dropped out of the rotation. Round 22's
// "Mongoose Bite and Lacerate swept at 33-60, the newest wins" compared the
// top rank against no cast at all. The spellbook keeps every rank; all
// Mongoose Bite ranks share one 5 s cooldown (category 5000 on every rank).
func TestMongooseBiteAndLacerateRegisterEveryLearnedRank(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              45,
			Equipment:          &proto.EquipmentSpec{},
			DistanceFromTarget: 5,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 30,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()

	built := sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()

	// Mongoose Bite ranks 1-3 are learned by 45 (16, 30, 44; rank 4 at 58).
	bites := map[int]int32{1: 1495, 2: 14269, 3: 14270}
	for rank, id := range bites {
		if built.GetSpell(core.ActionID{SpellID: id}) == nil {
			t.Errorf("Mongoose Bite rank %d (%d) is not registered at 45", rank, id)
		}
	}
	if built.GetSpell(core.ActionID{SpellID: 14271}) != nil {
		t.Errorf("Mongoose Bite rank 4 (14271) is registered at 45; it is learned at 58")
	}
	if built.MongooseBite == nil || built.MongooseBite.ActionID.SpellID != 14270 {
		t.Errorf("hunter.MongooseBite should stay the highest learned rank, 14270")
	}
	low, top := built.GetSpell(core.ActionID{SpellID: 1495}), built.GetSpell(core.ActionID{SpellID: 14270})
	if low != nil && top != nil && low.CD.Timer != top.CD.Timer {
		t.Errorf("Mongoose Bite ranks must share one cooldown (client category 5000 on every rank)")
	}

	// Lacerate ranks 1-2 are learned by 45 (30, 40; rank 3 at 50).
	for rank, id := range map[int]int32{1: 24118, 2: 24119} {
		if built.GetSpell(core.ActionID{SpellID: id}) == nil {
			t.Errorf("Lacerate rank %d (%d) is not registered at 45", rank, id)
		}
	}
	if built.Lacerate == nil || built.Lacerate.ActionID.SpellID != 24119 {
		t.Errorf("hunter.Lacerate should stay the highest learned rank, 24119")
	}
}
