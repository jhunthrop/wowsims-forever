package mage

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Counterspell trains at 24 (trainables 2139). It registered at every level
// with no cast-time level check, so a level 10 to 23 mage could cast it.
func TestCounterspellIsNotKnownBelowLevel24(t *testing.T) {
	for _, tc := range []struct {
		level int32
		known bool
	}{{23, false}, {24, true}} {
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
		if got := mage.Counterspell != nil; got != tc.known {
			t.Errorf("level %d: Counterspell registered = %v, want %v", tc.level, got, tc.known)
		}
	}
}
