package shaman_test

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/shaman"
)

// Tremor Totem trains at 18 (trainables 8143). It registered at every level
// with no cast-time level check, so a level 10 to 17 shaman could drop it.
func TestTremorTotemIsNotKnownBelowLevel18(t *testing.T) {
	for _, tc := range []struct {
		level int32
		known bool
	}{{17, false}, {18, true}} {
		player := core.WithSpec(
			&proto.Player{
				Class: proto.Class_ClassShaman, Race: proto.Race_RaceOrc, Level: tc.level,
				Equipment: &proto.EquipmentSpec{},
			},
			&proto.Player_EnhancementShaman{EnhancementShaman: &proto.EnhancementShaman{
				Options: &proto.EnhancementShaman_Options{SyncType: proto.ShamanSyncType_Auto},
			}},
		)
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, nil, nil, nil),
			Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{{Level: 63}}},
			SimOptions: &proto.SimOptions{RandomSeed: 1, Iterations: 1},
		}, simsignals.CreateSignals())
		sh := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent).GetShaman()
		if got := sh.TremorTotem != nil; got != tc.known {
			t.Errorf("level %d: Tremor Totem registered = %v, want %v", tc.level, got, tc.known)
		}
		for _, totem := range sh.EarthTotems {
			if totem == nil {
				t.Errorf("level %d: a nil spell in EarthTotems", tc.level)
			}
		}
	}
}
