package restoration

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// runUntil steps the simulation to the given time.
func runUntil(sim *core.Simulation, until time.Duration) {
	for sim.CurrentTime < until {
		if sim.Step() {
			return
		}
	}
}

// bloomHealing is what Bloom has healed the druid for.
func bloomHealing(resto *RestorationDruid) float64 {
	if resto.Bloom == nil {
		return 0
	}
	return resto.Bloom.SpellMetrics[resto.UnitIndex].TotalHealing
}

// Bloom (client 1.60.1.70707, spell 1328621): "When your Rejuvenation or
// Regrowth spell expires due to its duration, the target is instantly healed
// for an amount equal to 100% of your Spirit."
func TestBloomHealsForSpiritWhenRejuvenationRunsOut(t *testing.T) {
	for _, c := range []struct {
		name  string
		ranks map[string]int
		bloom bool
	}{
		{"untalented", nil, false},
		{"Bloom", map[string]int{"bloom": 1}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			resto, sim := healer(t, c.ranks)
			rejuvenation := resto.Rejuvenation[10]
			rejuvenation.ApplyEffects(sim, &resto.Unit, rejuvenation.Spell)
			spirit := resto.GetStat(stats.Spirit)
			runUntil(sim, 13*time.Second)

			got := bloomHealing(resto)
			if !c.bloom {
				if got != 0 {
					t.Errorf("Bloom healed %v without the talent", got)
				}
				return
			}
			if !near(got, spirit) {
				t.Errorf("Bloom healed %v when Rejuvenation ran out, want the druid's Spirit %v", got, spirit)
			}
		})
	}
}

// A Rejuvenation that Swiftmend consumes did not expire by its duration.
func TestBloomSkipsAConsumedRejuvenation(t *testing.T) {
	resto, sim := healer(t, map[string]int{"bloom": 1})
	rejuvenation := resto.Rejuvenation[10]
	rejuvenation.ApplyEffects(sim, &resto.Unit, rejuvenation.Spell)
	runUntil(sim, 3*time.Second)
	resto.Swiftmend.ApplyEffects(sim, &resto.Unit, resto.Swiftmend.Spell)
	if rejuvenation.Hot(&resto.Unit).IsActive() {
		t.Fatal("Swiftmend left the Rejuvenation up")
	}
	runUntil(sim, 13*time.Second)
	if got := bloomHealing(resto); got != 0 {
		t.Errorf("Bloom healed %v for a Rejuvenation Swiftmend consumed, want 0", got)
	}
}
