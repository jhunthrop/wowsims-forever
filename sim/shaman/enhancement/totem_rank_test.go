package enhancement

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/shaman"
)

func newUnbuffedShaman(t *testing.T, raid bool) (*core.Simulation, *shaman.Shaman) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassShaman,
			Race:      proto.Race_RaceOrc,
			Level:     60,
			Equipment: &proto.EquipmentSpec{},
		},
		&proto.Player_EnhancementShaman{
			EnhancementShaman: &proto.EnhancementShaman{
				Options: &proto.EnhancementShaman_Options{SyncType: proto.ShamanSyncType_Auto},
			},
		},
	)
	raidBuffs := &proto.RaidBuffs{}
	if raid {
		raidBuffs.StrengthOfEarthTotem = proto.TristateEffect_TristateEffectRegular
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, raidBuffs, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	return sim, agent.GetShaman()
}

// Round 9: a shaman's own Strength of Earth Totem gives its own rank's
// Strength. Rank 1 (8075, aura 8076) is 7 Strength in 1.60.1.70707; the
// engine gave every rank the level's top rank (42 at 60), so a rank 1 cast
// was the top rank's buff at the top rank's mana cost less 200.
func TestALowerStrengthOfEarthRankGivesItsOwnStrength(t *testing.T) {
	sim, built := newUnbuffedShaman(t, false)
	before := built.GetStat(stats.Strength)
	built.StrengthOfEarthTotem[1].Cast(sim, nil)
	if got := built.GetStat(stats.Strength) - before; got != 7 {
		t.Errorf("Strength of Earth Totem rank 1 gives %v Strength, want 7", got)
	}
}

// The raid's own Strength of Earth Totem (the level's rank) and the
// shaman's lower rank do not stack: the stronger one counts.
func TestALowerStrengthOfEarthRankDoesNotStackWithTheRaids(t *testing.T) {
	sim, built := newUnbuffedShaman(t, true)
	before := built.GetStat(stats.Strength)
	built.StrengthOfEarthTotem[1].Cast(sim, nil)
	if got := built.GetStat(stats.Strength) - before; got != 0 {
		t.Errorf("rank 1 under the raid's totem changes Strength by %v, want 0", got)
	}
}
