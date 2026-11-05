package mage

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Evocation trains at 20 (trainables 12051, spell_level 20). The engine
// registered it at every level and nothing checks RequiredLevel at cast
// time, so a level 10 to 19 mage channelled an Evocation the trainer has not
// taught (the ladder's level-10 rung cast one per fight).
func TestEvocationIsNotKnownBelowLevel20(t *testing.T) {
	for _, tc := range []struct {
		level int32
		known bool
	}{{19, false}, {20, true}} {
		player := core.WithSpec(&proto.Player{
			Class: proto.Class_ClassMage, Race: proto.Race_RaceTroll, Level: tc.level,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              &proto.IndividualBuffs{},
			DistanceFromTarget: 20,
			Rotation:           &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}, PlayerOptions)
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
			SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
		}, simsignals.CreateSignals())
		mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
		if got := mage.Evocation != nil; got != tc.known {
			t.Errorf("level %d: Evocation registered = %v, want %v", tc.level, got, tc.known)
		}
	}
}
