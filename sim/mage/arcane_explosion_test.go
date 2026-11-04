package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Arcane Explosion is a direct damage spell like Fire Blast or Cone of
// Cold: it rolls the spell hit table and can miss. The engine resolved it
// with OutcomeMagicCrit, a hit-only table, so it never missed.
func TestArcaneExplosionRollsTheSpellHitTable(t *testing.T) {
	built, sim, target := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, clearcastingTestFrostTalents)
	explosion := topRank(t, built.ArcaneExplosion)
	// Hit chance pinned to zero: every cast must miss.
	explosion.BonusHitRating -= 200 * core.HitRatingPerHitChance

	for i := 0; i < 5; i++ {
		sim.CurrentTime += 2 * time.Second
		built.GCD.Reset()
		if !explosion.Cast(sim, target) {
			t.Fatal("Arcane Explosion did not cast")
		}
		waitForOutcomes(sim)
	}
	metrics := explosion.SpellMetrics[target.UnitIndex]
	if metrics.Misses != 5 {
		t.Errorf("Arcane Explosion at zero hit missed %d of 5 casts, want 5: it never rolls the hit table", metrics.Misses)
	}
}
