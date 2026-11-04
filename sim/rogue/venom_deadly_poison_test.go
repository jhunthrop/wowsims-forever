package rogue_test

import (
	"math"
	"testing"
)

// Venom (1310703) is a 6-second buff from Shank: +30% poison damage, added
// to Vile Poisons. A five-stack Deadly Poison keeps being re-applied by new
// procs, and every re-application takes the caster's modifiers of that
// moment. The poison used to keep the multiplier of its FIRST stack for
// as long as it never dropped, so a fight whose first stack landed under
// Venom kept Venom's 30% on every tick for the whole fight, and one whose
// first stack landed outside it never got it (sim-audit round 18).
func TestDeadlyPoisonReapplicationTakesVenomOfTheMoment(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, talentsWithRanks(t, map[string]int{"venom": 1, "vile_poisons": 5}), "combat_backstab_prebis")
	rogue := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]
	if rogue.VenomAura == nil || rogue.DeadlyPoison == nil {
		t.Fatal("a level 60 rogue with Venom has no Venom aura or Deadly Poison")
	}

	stack := func(n int) {
		for i := 0; i < n; i++ {
			rogue.DeadlyPoison.Cast(sim, target)
		}
	}
	stack(20)
	if got := rogue.DeadlyPoisonStacksForTest(target); got != 5 {
		t.Fatalf("Deadly Poison at %d stacks after twenty applications, want 5", got)
	}
	plain := rogue.DeadlyPoisonSnapshotMultiplierForTest(target)

	rogue.VenomAura.Activate(sim)
	stack(5)
	withVenom := rogue.DeadlyPoisonSnapshotMultiplierForTest(target)
	// Vile Poisons 5 is 1.20; Venom adds 0.30 to it.
	if want := plain * 1.50 / 1.20; math.Abs(withVenom-want) > 1e-9 {
		t.Errorf("a re-application under Venom keeps multiplier %.4f, want %.4f (Venom's 30%% added to Vile Poisons' 20%%)", withVenom, want)
	}

	rogue.VenomAura.Deactivate(sim)
	stack(5)
	if after := rogue.DeadlyPoisonSnapshotMultiplierForTest(target); math.Abs(after-plain) > 1e-9 {
		t.Errorf("a re-application after Venom ends keeps multiplier %.4f, want %.4f", after, plain)
	}
}
