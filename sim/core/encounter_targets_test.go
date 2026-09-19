package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// The pool is sized to the largest count the timeline asks for, by
// repeating the last configured target. The site sends one target and a
// timeline; it does not send five copies.
func TestTimelineSizesThePoolToItsLargestCount(t *testing.T) {
	enc := NewEncounter(&proto.Encounter{
		Duration: 180,
		Targets:  []*proto.Target{DefaultTargetProtoLvl60},
		TargetsOverTime: []*proto.TargetCountAt{
			{AtSeconds: 0, Count: 1},
			{AtSeconds: 40, Count: 5},
			{AtSeconds: 120, Count: 3},
		},
	})
	if got := len(enc.AllTargetUnits); got != 5 {
		t.Errorf("pool size = %d, want 5", got)
	}
	if got := len(enc.TargetUnits); got != 1 {
		t.Errorf("active count at construction = %d, want the timeline's t=0 count of 1", got)
	}
}

// Entries arrive in whatever order the request had them; the engine sorts.
func TestTimelineIsSortedByTime(t *testing.T) {
	enc := NewEncounter(&proto.Encounter{
		Duration: 180,
		Targets:  []*proto.Target{DefaultTargetProtoLvl60},
		TargetsOverTime: []*proto.TargetCountAt{
			{AtSeconds: 120, Count: 3},
			{AtSeconds: 0, Count: 1},
			{AtSeconds: 40, Count: 5},
		},
	})
	want := []TargetCount{
		{At: 0, Count: 1},
		{At: 40 * time.Second, Count: 5},
		{At: 120 * time.Second, Count: 3},
	}
	if len(enc.TargetsOverTime) != len(want) {
		t.Fatalf("TargetsOverTime = %v, want %v", enc.TargetsOverTime, want)
	}
	for i, tc := range want {
		if enc.TargetsOverTime[i] != tc {
			t.Errorf("TargetsOverTime[%d] = %+v, want %+v", i, enc.TargetsOverTime[i], tc)
		}
	}
}

// An encounter with no timeline is exactly today's encounter.
func TestNoTimelineLeavesEveryTargetActive(t *testing.T) {
	enc := NewEncounter(&proto.Encounter{
		Duration: 180,
		Targets:  []*proto.Target{DefaultTargetProtoLvl60, DefaultTargetProtoLvl60},
	})
	if len(enc.TargetsOverTime) != 0 {
		t.Errorf("TargetsOverTime = %v, want empty", enc.TargetsOverTime)
	}
	if got := len(enc.TargetUnits); got != 2 {
		t.Errorf("active count = %d, want both targets", got)
	}
}

// Activating and deactivating is what the timeline does at run time: the
// active prefix grows and shrinks, GetNumTargets follows it, and the AoE
// cap multiplier is recomputed against the active count, not the pool.
func TestSetActiveTargetCountMovesThePrefix(t *testing.T) {
	sim := timelineSim(t, []*proto.TargetCountAt{{AtSeconds: 0, Count: 1}, {AtSeconds: 40, Count: 3}})

	if got := sim.GetNumTargets(); got != 1 {
		t.Fatalf("GetNumTargets after reset = %d, want 1", got)
	}
	if sim.Encounter.Targets[2].IsEnabled() {
		t.Error("target 3 is enabled while the timeline says one target")
	}

	sim.Encounter.SetActiveTargetCount(sim, 3)
	if got := sim.GetNumTargets(); got != 3 {
		t.Errorf("GetNumTargets after activating = %d, want 3", got)
	}
	if !sim.Encounter.Targets[2].IsEnabled() {
		t.Error("target 3 is disabled after activating three")
	}
	if got := sim.Encounter.AOECapMultiplier(); got != 1 {
		t.Errorf("AOECapMultiplier with 3 active = %v, want 1", got)
	}

	sim.Encounter.SetActiveTargetCount(sim, 1)
	if got := sim.GetNumTargets(); got != 1 {
		t.Errorf("GetNumTargets after deactivating = %d, want 1", got)
	}
	if sim.Encounter.Targets[2].IsEnabled() {
		t.Error("target 3 is enabled after deactivating back to one")
	}
}

// A count below one or above the pool is clamped rather than panicking:
// the request is validated on the site, and the engine is not the place
// to crash a paying run over a bad number.
func TestSetActiveTargetCountClamps(t *testing.T) {
	sim := timelineSim(t, []*proto.TargetCountAt{{AtSeconds: 0, Count: 1}, {AtSeconds: 40, Count: 3}})

	sim.Encounter.SetActiveTargetCount(sim, 0)
	if got := sim.GetNumTargets(); got != 1 {
		t.Errorf("GetNumTargets after asking for 0 = %d, want the clamped 1", got)
	}
	sim.Encounter.SetActiveTargetCount(sim, 99)
	if got := sim.GetNumTargets(); got != 3 {
		t.Errorf("GetNumTargets after asking for 99 = %d, want the clamped pool size 3", got)
	}
}

// A player whose current target is deactivated is retargeted, or its
// rotation spends the rest of the fight attacking a corpse.
func TestDeactivatingRetargetsThePlayer(t *testing.T) {
	sim := timelineSim(t, []*proto.TargetCountAt{{AtSeconds: 0, Count: 3}, {AtSeconds: 40, Count: 1}})
	player := &sim.Raid.Parties[0].Players[0].GetCharacter().Unit
	player.CurrentTarget = sim.Encounter.AllTargetUnits[2]

	sim.Encounter.SetActiveTargetCount(sim, 1)

	if player.CurrentTarget != sim.Encounter.AllTargetUnits[0] {
		t.Errorf("CurrentTarget = %v after its target went away, want the first active target", player.CurrentTarget.Label)
	}
}

// timelineSim builds a one-player simulation with the given timeline,
// reset and ready to step.
func timelineSim(t *testing.T, timeline []*proto.TargetCountAt) *Simulation {
	t.Helper()
	sim := NewSim(&proto.RaidSimRequest{
		Raid: SinglePlayerRaidProto(&proto.Player{
			Name:      "Timeline Test",
			Race:      proto.Race_RaceOrc,
			Class:     proto.Class_ClassShaman,
			Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment: &proto.EquipmentSpec{},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration:        180,
			Targets:         []*proto.Target{DefaultTargetProtoLvl60},
			TargetsOverTime: timeline,
		},
		SimOptions: &proto.SimOptions{Iterations: 1, IsTest: true, RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.reset()
	return sim
}
