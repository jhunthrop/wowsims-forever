package dpswarrior

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/warrior"
)

// Blizzard, forever-bugs #234 (2026-10-05): Deep Wounds "Doesn't reset its
// tick timer when it is refreshed". A refresh 4 s after the first
// application (one tick in, at 3 s) keeps the 3 s phase: the ticks land at
// 3, 6, 9, 12 and 15 s, four of them from the refresh, and no extra tick
// comes out of the refresh itself.
func TestDeepWoundsRefreshKeepsTheTickTimer(t *testing.T) {
	sim, war := newWarriorSim(t, warrior.ForeverFuryTalents)
	if war.DeepWounds == nil {
		t.Fatal("the Fury build should train Deep Wounds")
	}
	war.AutoAttacks.CancelAutoSwing(sim)

	target := sim.Encounter.TargetUnits[0]
	deepWounds := war.DeepWounds
	start := sim.CurrentTime
	deepWounds.Dot(target).SnapshotBaseDamage = 100
	deepWounds.Cast(sim, target)
	sim.AddPendingAction(&core.PendingAction{
		NextActionAt: start + 4*time.Second,
		OnAction: func(sim *core.Simulation) {
			deepWounds.Cast(sim, target)
		},
	})

	var tickTimes []time.Duration
	var seen int32
	for sim.CurrentTime < start+30*time.Second && !sim.Step() {
		m := deepWounds.SpellMetrics[target.UnitIndex]
		for ; seen < m.Ticks+m.CritTicks; seen++ {
			tickTimes = append(tickTimes, sim.CurrentTime-start)
		}
	}

	want := []time.Duration{3 * time.Second, 6 * time.Second, 9 * time.Second, 12 * time.Second, 15 * time.Second}
	if len(tickTimes) != len(want) {
		t.Fatalf("ticks at %v, want %v", tickTimes, want)
	}
	for i := range want {
		if tickTimes[i] != want[i] {
			t.Errorf("ticks at %v, want %v", tickTimes, want)
			break
		}
	}
}
