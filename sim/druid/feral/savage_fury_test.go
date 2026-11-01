package feral

import "testing"

// Savage Fury (16998) raises Claw, Rake, Shred, Maul and Swipe: its client
// mask 0x9800 holds Shred (0x8000), Claw and Rake (0x1000), and Maul and
// Swipe (0x800). Pounce (0x20000) is not in it. The 2026-10-15 beta notes
// say Shred's damage already counted the talent.
func TestSavageFuryRaisesShredAndNotPounce(t *testing.T) {
	built, _, _ := newFeralDruidSimAtLevel(t, 60)
	if built.Talents.SavageFury == 0 {
		t.Fatal("P1Talents takes no Savage Fury")
	}
	want := built.Claw.DamageMultiplierAdditive
	if want <= 1 {
		t.Fatalf("Claw's additive multiplier %v carries no Savage Fury", want)
	}
	if got := built.Shred.DamageMultiplierAdditive; got != want {
		t.Errorf("Shred's additive multiplier %v, want Savage Fury's %v", got, want)
	}
	if got := built.Pounce.DamageMultiplierAdditive; got != 1 {
		t.Errorf("Pounce's additive multiplier %v, want 1: Savage Fury does not cover it", got)
	}
}
