package dpswarrior

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/warrior"
)

// Deep Wounds' ticks never crit: the client marks Forever's Deep Wound
// (412613) Can't Crit, and the launch logs hold 0 critical ticks in 146
// where the same warriors' Rend ticks crit at their melee crit chance. It
// is the one bleed outside the owner's periodic crit rule. A tick must not
// refresh Deep Wounds either: one application runs its four ticks and ends.
func TestDeepWoundsTicksNeverCrit(t *testing.T) {
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
	if m.CritTicks != 0 || m.Ticks != ticks {
		t.Errorf("Deep Wounds ticked %d plain and %d critical at 200%% crit, want %d plain and none", m.Ticks, m.CritTicks, ticks)
	}
	if m.Casts != 1 {
		t.Errorf("Deep Wounds was applied %d times from one forced cast, want 1", m.Casts)
	}
}
