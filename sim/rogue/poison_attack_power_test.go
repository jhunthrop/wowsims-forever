package rogue_test

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/rogue"
)

// Forever's poisons scale with attack power. Blizzard's beta development
// notes of 3 November 2026 (the changes since the 15 October update): "Deadly
// Poison Attack Power coefficient increased to 0.009 (Was 0.0045)" and
// "Instant Poison Attack Power coefficient increased to 0.01 (Was 0.005)".
// The client's SpellEffect rows carry 0 in BonusCoefficientFromAP for both,
// as they do for Garrote's vanilla 0.03 a tick: these coefficients live on
// the server. The engine's poisons scaled with nothing (sim-audit round 20).

// Each Deadly Poison stack's tick is its client amount plus 0.009 of the
// rogue's attack power, read when the stack is applied.
func TestDeadlyPoisonTicksScaleWithAttackPower(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, talentsWithRanks(t, map[string]int{}), "combat_backstab_prebis")
	r := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]
	if r.DeadlyPoison == nil {
		t.Fatal("a level 60 rogue has no Deadly Poison")
	}

	for i := 0; i < 40 && r.DeadlyPoisonStacksForTest(target) < 5; i++ {
		r.DeadlyPoison.Cast(sim, target)
	}
	if got := r.DeadlyPoisonStacksForTest(target); got != 5 {
		t.Fatalf("Deadly Poison at %d stacks, want 5", got)
	}

	ap := r.PoisonAttackPowerForTest(target)
	if ap < 500 {
		t.Fatalf("the fixture rogue has %.0f attack power, too little to tell the coefficient from the roll", ap)
	}
	tick := rogue.DeadlyPoisonTickDamage[5].Center(60)
	want := 5 * (tick + 0.009*ap)
	if got := r.DeadlyPoisonSnapshotBaseDamageForTest(target); math.Abs(got-want) > 1e-6 {
		t.Errorf("five Deadly Poison stacks tick for %.2f, want %.2f (5 x (%.0f + 0.009 x %.0f attack power))", got, want, tick, ap)
	}
}

// An Instant Poison hit is its client roll plus 0.01 of the rogue's attack
// power.
func TestInstantPoisonHitsScaleWithAttackPower(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, talentsWithRanks(t, map[string]int{}), "combat_backstab_prebis")
	r := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]
	if r.InstantPoison == nil {
		t.Fatal("a level 60 rogue has no Instant Poison")
	}

	ap := r.PoisonAttackPowerForTest(target)
	if ap < 500 {
		t.Fatalf("the fixture rogue has %.0f attack power, too little to tell the coefficient from the roll", ap)
	}
	roll := rogue.InstantPoisonDamage[6].Range(60)
	lo, hi := roll[0]+0.01*ap, roll[1]+0.01*ap
	for i := 0; i < 200; i++ {
		got := r.InstantPoisonBaseDamageForTest(sim, target)
		if got < lo-1e-9 || got > hi+1e-9 {
			t.Fatalf("an Instant Poison hit of %.2f is outside %.2f to %.2f (the %.0f to %.0f roll plus 0.01 x %.0f attack power)", got, lo, hi, roll[0], roll[1], ap)
		}
	}
}
