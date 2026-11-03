package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

// manaTickSim is manaConsumableSim's character with a mana bar wired in (the
// core test agent never enables one), so the regen tick is scheduled.
func manaTickSim(t *testing.T) *Simulation {
	t.Helper()
	sim, character := manaConsumableSim(t, nil)
	character.manaBar.unit = &character.Unit
	character.manaCastingMetrics = character.NewManaMetrics(ActionID{OtherID: proto.OtherAction_OtherActionManaRegen, Tag: 1})
	character.manaNotCastingMetrics = character.NewManaMetrics(ActionID{OtherID: proto.OtherAction_OtherActionManaRegen, Tag: 2})
	return sim
}

// firstManaTickOffset resets the sim at an iteration seed and returns how
// long after the prepull start the first regen tick is due.
func firstManaTickOffset(t *testing.T, sim *Simulation, iteration int64) time.Duration {
	t.Helper()
	sim.Cleanup()
	sim.reseedRands(iteration)
	sim.reset()
	for _, pa := range sim.pendingActions {
		if pa.Priority == ActionPriorityRegen {
			return pa.NextActionAt - sim.Environment.PrepullStartTime()
		}
	}
	t.Fatal("no mana regen tick is scheduled")
	return 0
}

// A server's 2 s regen tick runs on its own clock, so its phase against the
// pull is arbitrary (owner ruling, round 13). Each iteration draws the first
// tick from [0, 2 s) after the prepull start; the same seed draws the same
// phase.
func TestManaTickPhaseIsRandomPerIteration(t *testing.T) {
	sim := manaTickSim(t)
	const iterations = 200
	period := time.Duration(manaTickInterval * float64(time.Second))

	lo, hi := period, time.Duration(0)
	seen := map[time.Duration]bool{}
	for i := int64(0); i < iterations; i++ {
		offset := firstManaTickOffset(t, sim, i)
		if offset < 0 || offset >= period {
			t.Fatalf("iteration %d: first tick %v after the prepull start, want within [0, %v)", i, offset, period)
		}
		lo, hi = min(lo, offset), max(hi, offset)
		seen[offset] = true
	}
	if len(seen) < iterations*9/10 {
		t.Errorf("%d distinct first-tick phases over %d iterations, want nearly all distinct", len(seen), iterations)
	}
	if lo > period/10 || hi < period*9/10 {
		t.Errorf("first-tick phases span [%v, %v], want close to [0, %v)", lo, hi, period)
	}

	for _, i := range []int64{0, 7, 123} {
		first := firstManaTickOffset(t, sim, i)
		if again := firstManaTickOffset(t, sim, i); again != first {
			t.Errorf("iteration %d drew phase %v, then %v on a rerun of the same seed", i, first, again)
		}
	}
}

// The grid keeps its 2 s period after the drawn phase.
func TestManaTicksStayTwoSecondsApart(t *testing.T) {
	sim := manaTickSim(t)
	firstManaTickOffset(t, sim, 3)
	var regen *PendingAction
	for _, pa := range sim.pendingActions {
		if pa.Priority == ActionPriorityRegen {
			regen = pa
		}
	}
	first := regen.NextActionAt
	sim.CurrentTime = first
	regen.OnAction(sim)
	if got := regen.NextActionAt - first; got != 2*time.Second {
		t.Errorf("second tick %v after the first, want 2s", got)
	}
}
