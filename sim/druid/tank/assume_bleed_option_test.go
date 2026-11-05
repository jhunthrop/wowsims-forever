package tank

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// The bear's options carry assume_bleed_active as the cat's do (Rend and
// Tear on a target another raider keeps bleeding), and it reaches the druid.
func TestBearAssumeBleedActiveOptionReachesTheDruid(t *testing.T) {
	for _, want := range []bool{true, false} {
		player := core.WithSpec(&proto.Player{
			Class: proto.Class_ClassDruid, Race: proto.Race_RaceTauren, Level: 60,
			Equipment: &proto.EquipmentSpec{}, DistanceFromTarget: 5,
		}, &proto.Player_FeralTankDruid{FeralTankDruid: &proto.FeralTankDruid{Options: &proto.FeralTankDruid_Options{
			InnervateTarget: &proto.UnitReference{}, AssumeBleedActive: want,
		}}})
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  harnessEncounter(),
			SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
		}, simsignals.CreateSignals())
		bear := sim.Raid.Parties[0].Players[0].(*FeralTankDruid)
		if bear.AssumeBleedActive != want {
			t.Errorf("options assume_bleed_active %v: Druid.AssumeBleedActive = %v", want, bear.AssumeBleedActive)
		}
	}
}
