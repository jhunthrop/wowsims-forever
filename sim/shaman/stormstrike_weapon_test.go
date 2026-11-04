package shaman_test

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/shaman"
	"github.com/wowsims/classic/sim/shaman/enhancement"
)

// enhancementRaidTalents is the published level-60 raid Enhancement build
// (Stormstrike 1/1).
func init() {
	enhancement.RegisterEnhancementShaman()
}

const enhancementRaidTalents = "0533001000000000-253030310051002051-0530000000000000"

func enhancementSim(t *testing.T) (*core.Simulation, *shaman.Shaman) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassShaman,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     core.GetGearSet("../../ui/enhancement_shaman/gear_sets", "phase_5").GearSet,
			TalentsString: enhancementRaidTalents,
		},
		&proto.Player_EnhancementShaman{
			EnhancementShaman: &proto.EnhancementShaman{
				Options: &proto.EnhancementShaman_Options{SyncType: proto.ShamanSyncType_Auto},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, nil, nil, nil)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{{Level: 63}}},
		SimOptions: &proto.SimOptions{RandomSeed: 42, Iterations: 1},
	}, simsignals.CreateSignals())
	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	return sim, agent.GetShaman()
}

// Stormstrike 17364's effect 0 is client effect 121 (normalized weapon
// damage, 1.60.1.70707 SpellEffect): its attack power counts at the weapon
// type's normalized speed (3.3 s for a two-hander, 2.4 s for a one-hander),
// not the weapon's own swing speed, as Hemorrhage, Backstab and Sinister
// Strike do. Effect 2, added in 1.60.1.70755, is Effect 31 (weapon percent
// damage) at 150: the strike is 150% of that normalized weapon damage.
func TestStormstrikeUsesNormalizedWeaponDamage(t *testing.T) {
	const ap = 1500.0
	sim, built := enhancementSim(t)
	if built.Stormstrike == nil {
		t.Fatal("the raid build has no Stormstrike")
	}
	got := built.StormstrikeWeaponDamageForTest(sim, ap)
	refSim, ref := enhancementSim(t)
	mh := ref.AutoAttacks.MH()
	if mh.SwingSpeed == mh.NormalizedSwingSpeed {
		t.Fatalf("the test weapon swings at its normalized %.1f s, so it cannot tell the two apart", mh.SwingSpeed)
	}
	want := 1.5 * mh.CalculateNormalizedWeaponDamage(refSim, ap)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Stormstrike weapon damage = %.3f, want 150%% of the normalized, %.3f (weapon %.2f s)", got, want, mh.SwingSpeed)
	}
	if got, want := built.Stormstrike.CD.Duration, 6*time.Second; got != want {
		t.Errorf("Stormstrike cooldown = %v, want %v (client RecoveryTime 6000)", got, want)
	}
}
