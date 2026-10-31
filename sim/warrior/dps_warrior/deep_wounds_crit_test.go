package dpswarrior

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/warrior"
)

// Forever's Deep Wounds ticks crit (the owner's periodic crit rule: every
// bleed crits), on the melee table with the physical multiplier. A tick
// crit must not refresh Deep Wounds: the talent's proc mask (spell 12834,
// SpellAuraOptions ProcTypeMask 0x11154) has no periodic bit, so one
// application runs its four ticks and ends.
func TestDeepWoundsTicksCritWithoutRefreshing(t *testing.T) {
	sim, war := newWarriorSim(t, warrior.ForeverFuryTalents)
	if war.DeepWounds == nil {
		t.Fatal("the Fury build should train Deep Wounds")
	}
	// No swings: only the forced application below may start the bleed.
	war.AutoAttacks.CancelAutoSwing(sim)

	target := sim.Encounter.TargetUnits[0]
	deepWounds := war.DeepWounds
	deepWounds.BonusCritRating += 200 * core.CritRatingPerCritChance
	deepWounds.Dot(target).SnapshotBaseDamage = 100
	deepWounds.Cast(sim, target)

	const ticks = 4
	deadline := sim.CurrentTime + 3*ticks*time.Second + time.Second
	for sim.CurrentTime < deadline && !sim.Step() {
	}

	m := deepWounds.SpellMetrics[target.UnitIndex]
	if m.CritTicks != ticks {
		t.Errorf("Deep Wounds CritTicks = %d (plain %d), want %d at 100%% crit", m.CritTicks, m.Ticks, ticks)
	}
	if m.Casts != 1 || m.CritTicks+m.Ticks != ticks {
		t.Errorf("Deep Wounds tick crits refreshed it: %d casts, %d ticks, want 1 cast and %d ticks", m.Casts, m.CritTicks+m.Ticks, ticks)
	}
}
