package feral

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// The Feral options' assume_bleed_active ("the raid keeps a bleed on the
// target") must reach Druid.AssumeBleedActive, which Rend and Tear reads.
// NewFeralDruid never copied it, so the option did nothing.
func TestAssumeBleedActiveOptionReachesTheDruid(t *testing.T) {
	for _, tc := range []struct {
		options *proto.Player_FeralDruid
		want    bool
	}{{PlayerOptionsMonoCat, true}, {PlayerOptionsMonoCatNoBleed, false}} {
		player := core.WithSpec(&proto.Player{
			Class: proto.Class_ClassDruid, Race: proto.Race_RaceTauren, Level: 60,
			Equipment: &proto.EquipmentSpec{}, DistanceFromTarget: 5,
		}, tc.options)
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
			SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
		}, simsignals.CreateSignals())
		cat := sim.Raid.Parties[0].Players[0].(*FeralDruid)
		if cat.AssumeBleedActive != tc.want {
			t.Errorf("options assume_bleed_active %v: Druid.AssumeBleedActive = %v", tc.want, cat.AssumeBleedActive)
		}
	}
}
