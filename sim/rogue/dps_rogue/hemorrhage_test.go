package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's Hemorrhage (spellconst/rogue.json, 16511): "causes the target to
// take 15% increased Rupture damage from the Rogue. Lasts 15 sec." It is no
// longer the vanilla "+7 physical damage on the next 30 hits" debuff.
func TestHemorrhageBoostsRuptureNotPhysicalHits(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t, "hemorrhage"))
	if built.Hemorrhage == nil {
		t.Fatal("talented rogue has no Hemorrhage")
	}
	target := sim.Encounter.TargetUnits[0]

	if got := built.RuptureDamageTakenMultiplier(target); got != 1 {
		t.Fatalf("Rupture multiplier before Hemorrhage = %v, want 1", got)
	}

	for i := 0; i < 50 && built.RuptureDamageTakenMultiplier(target) == 1; i++ {
		built.Hemorrhage.ApplyEffects(sim, target, built.Hemorrhage)
	}
	if got, want := built.RuptureDamageTakenMultiplier(target), 1.15; got != want {
		t.Errorf("Rupture multiplier after a landed Hemorrhage = %v, want %v", got, want)
	}
	if got := target.PseudoStats.SchoolBonusDamageTaken[stats.SchoolIndexPhysical]; got != 0 {
		t.Errorf("Hemorrhage added %v flat physical damage taken, want none", got)
	}
}
