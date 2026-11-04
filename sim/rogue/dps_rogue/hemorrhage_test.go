package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's Hemorrhage (16511, 70755): "causes the target to take 15%
// increased Rupture and Garrote damage from the Rogue" (e2 aura 271 bp 15
// on mask 1048832). It is not the vanilla "+7 physical damage" debuff.
func TestHemorrhageBoostsRuptureNotPhysicalHits(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t, "hemorrhage"))
	if built.Hemorrhage == nil {
		t.Fatal("talented rogue has no Hemorrhage")
	}
	target := sim.Encounter.TargetUnits[0]

	if got := built.BleedDamageTakenMultiplier(target); got != 1 {
		t.Fatalf("Rupture multiplier before Hemorrhage = %v, want 1", got)
	}

	for i := 0; i < 50 && built.BleedDamageTakenMultiplier(target) == 1; i++ {
		built.Hemorrhage.ApplyEffects(sim, target, built.Hemorrhage)
	}
	if got, want := built.BleedDamageTakenMultiplier(target), 1.15; got != want {
		t.Errorf("Rupture multiplier after a landed Hemorrhage = %v, want %v", got, want)
	}
	if got := target.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexPhysical]; got != 0 {
		t.Errorf("Hemorrhage added %v flat physical damage taken, want none", got)
	}
}
