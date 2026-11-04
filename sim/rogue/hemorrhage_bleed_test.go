package rogue_test

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Hemorrhage (16511) leaves a debuff: the target takes 15% more Rupture and
// Garrote damage from the rogue for 15 seconds, read at each tick. The
// bleed tick scaled the spell's live multiplier around the tick, but a
// snapshot tick never reads it (it uses the multiplier taken at the cast),
// so the 15% never landed (sim-audit round 18).
func TestHemorrhageRaisesRuptureAndGarroteTicks(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, talentsWithRanks(t, map[string]int{"hemorrhage": 1}), "combat_backstab_prebis")
	rogue := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]
	if rogue.Hemorrhage == nil {
		t.Fatal("a level 60 rogue with the talent has no Hemorrhage")
	}

	for _, bleed := range []*core.Spell{rogue.Rupture, rogue.Garrote} {
		dot := bleed.Dot(target)
		dot.Snapshot(target, 100, false)
		plain := rogue.BleedTickDamageForTest(sim, target, bleed)
		rogue.HemorrhageDebuffForTest(target).Activate(sim)
		raised := rogue.BleedTickDamageForTest(sim, target, bleed)
		rogue.HemorrhageDebuffForTest(target).Deactivate(sim)
		if plain <= 0 {
			t.Fatalf("%s tick dealt %v", bleed.ActionID, plain)
		}
		if got := raised / plain; math.Abs(got-1.15) > 1e-9 {
			t.Errorf("%s tick under Hemorrhage is x%.4f of the plain tick, want x1.15", bleed.ActionID, got)
		}
	}
}
