package mage

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// TestIceLanceFrozenMultiplierIsFour pins client 1.60.1.70707's talent
// text: "Deals 300% increased damage to Frozen targets" is +300%, so x4
// against a Frozen target (x3 from 70425 through 70685).
func TestIceLanceFrozenMultiplierIsFour(t *testing.T) {
	if iceLanceFrozenMultiplier != 4.0 {
		t.Errorf("iceLanceFrozenMultiplier = %v, want 4 (+300%%)", iceLanceFrozenMultiplier)
	}
}

// TestIceLanceFrozenBonusMultipliesTheWholeHit: "Deals 300% increased
// damage to Frozen targets" is damage, not the base roll, so the spell
// power share is multiplied too. With 1000 spell power the share
// (0.4286 a point) outweighs the base, so multiplying only the roll
// would leave a Frozen Lance under twice an unfrozen one.
func TestIceLanceFrozenBonusMultipliesTheWholeHit(t *testing.T) {
	average := func(frozen bool) float64 {
		sim, built := newIgniteMageSim(t)
		target := sim.Encounter.TargetUnits[0]
		lance := built.IceLance[len(built.IceLance)-1]
		if lance == nil || built.FrozenAuras == nil {
			t.Fatal("level-60 mage has no top-rank Ice Lance or no Frost Nova Frozen aura")
		}
		built.AddStatDynamic(sim, stats.SpellPower, 1000)
		lance.BonusCritRating -= 1000 * core.CritRatingPerCritChance
		lance.BonusHitRating += 100 * core.HitRatingPerHitChance
		for i := 0; i < 20; i++ {
			if frozen {
				built.FrozenAuras.Get(target).Activate(sim)
			}
			lance.ApplyEffects(sim, target, lance)
			built.FrozenAuras.Get(target).Deactivate(sim)
			waitForOutcomes(sim)
		}
		waitForOutcomes(sim)
		m := lance.SpellMetrics[target.UnitIndex]
		return m.TotalDamage / float64(m.Hits)
	}
	plain := average(false)
	frozen := average(true)
	if ratio := frozen / plain; ratio < 3.8 || ratio > 4.2 {
		t.Errorf("a Frozen Ice Lance dealt %.2fx an unfrozen one (%.1f against %.1f), want 4x", ratio, frozen, plain)
	}
}
