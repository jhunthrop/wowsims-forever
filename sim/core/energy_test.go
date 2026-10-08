package core

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

const energyTolerance = 0.1

// newTestEnergyBar returns an empty, running energy bar and the sim whose
// clock drives it. No APL is attached, so only the accrual maths is under test.
func newTestEnergyBar(t *testing.T, maxEnergy float64) (*energyBar, *Simulation) {
	t.Helper()
	char := newTestCharacter(t, 60, proto.Class_ClassRogue, proto.Race_RaceHuman)
	char.EnableEnergyBar(maxEnergy)
	sim := &Simulation{}
	bar := &char.energyBar
	bar.sim = sim
	bar.enable(sim, 0)
	bar.currentEnergy = 0
	return bar, sim
}

func assertEnergy(t *testing.T, bar *energyBar, want float64) {
	t.Helper()
	if got := bar.CurrentEnergy(); math.Abs(got-want) > energyTolerance {
		t.Fatalf("energy = %v, want %v", got, want)
	}
}

func TestEnergyRegeneratesTenPerSecond(t *testing.T) {
	bar, sim := newTestEnergyBar(t, 100)
	sim.CurrentTime = time.Second
	assertEnergy(t, bar, 10)
}

func TestEnergyRegeneratesContinuouslyBetweenTicks(t *testing.T) {
	bar, sim := newTestEnergyBar(t, 100)
	sim.CurrentTime = 350 * time.Millisecond
	assertEnergy(t, bar, 3.5)
}

func TestEnergyNeverExceedsMaximum(t *testing.T) {
	bar, sim := newTestEnergyBar(t, 110)
	sim.CurrentTime = time.Minute
	assertEnergy(t, bar, 110)
}

func TestAdrenalineRushDoublesTheRate(t *testing.T) {
	bar, sim := newTestEnergyBar(t, 100)
	sim.CurrentTime = time.Second
	bar.AddEnergyRegenMultiplier(1)
	sim.CurrentTime = 2 * time.Second
	assertEnergy(t, bar, 10+20)
	bar.AddEnergyRegenMultiplier(-1)
	sim.CurrentTime = 3 * time.Second
	assertEnergy(t, bar, 10+20+10)
}

func TestSpendKeepsFractionalAccrual(t *testing.T) {
	bar, sim := newTestEnergyBar(t, 100)
	sim.CurrentTime = 350 * time.Millisecond
	bar.SpendEnergy(sim, 1, bar.regenMetrics)
	assertEnergy(t, bar, 2.5)
	sim.CurrentTime = 450 * time.Millisecond
	assertEnergy(t, bar, 3.5)
}

func TestEnergyTimeConversionsRoundTrip(t *testing.T) {
	if got := EnergyForTime(2020 * time.Millisecond); math.Abs(got-20.2) > 1e-9 {
		t.Fatalf("EnergyForTime(2.02s) = %v, want 20.2", got)
	}
	if got := TimeForEnergy(20.2); got != 2020*time.Millisecond {
		t.Fatalf("TimeForEnergy(20.2) = %v, want 2.02s", got)
	}
}

func TestWakeLandsOnTheNextDecisionThreshold(t *testing.T) {
	bar, sim := newTestEnergyBar(t, 100)
	bar.unit.Rotation = &APLRotation{}
	bar.energyDecisionThresholds = []int{10, 40}
	bar.cumulativeEnergyDecisionThresholds = make([]int, 101)

	sim.CurrentTime = 350 * time.Millisecond // 3.5 energy, 6.5 short of 10
	bar.accrue()
	if got, want := bar.computeWakeAt()-sim.CurrentTime, 650*time.Millisecond; got < want || got > want+time.Microsecond {
		t.Fatalf("wake in %v, want %v", got, want)
	}

	bar.AddEnergyRegenMultiplier(1) // Adrenaline Rush halves the wait
	if got, want := bar.computeWakeAt()-sim.CurrentTime, 325*time.Millisecond; got < want || got > want+time.Microsecond {
		t.Fatalf("wake in %v with double regen, want %v", got, want)
	}
}

func TestWakeNeverWhenFullOrWithoutAPL(t *testing.T) {
	bar, sim := newTestEnergyBar(t, 100)
	if bar.computeWakeAt() != NeverExpires {
		t.Fatal("a bar with no APL should never wake")
	}
	bar.unit.Rotation = &APLRotation{}
	sim.CurrentTime = time.Minute
	bar.accrue()
	if bar.computeWakeAt() != NeverExpires {
		t.Fatal("a full bar should never wake")
	}
}
