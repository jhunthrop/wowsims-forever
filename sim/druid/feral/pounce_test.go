package feral

import (
	"testing"
	"time"
)

// Pounce needs Prowl, like Ravage: the client's text says "Must be
// prowling and behind the target" and carries no flag for it.
func TestPounceOnlyCastableWhileProwling(t *testing.T) {
	built, sim, target := newFeralDruidSimAtLevel(t, 60)

	if built.Pounce == nil {
		t.Fatal("level-60 Feral druid has no Pounce registered")
	}
	if got, want := built.Pounce.ActionID.SpellID, int32(9827); got != want {
		t.Errorf("Pounce spell ID = %d, want max rank %d", got, want)
	}
	if built.Pounce.CanCast(sim, target) {
		t.Fatal("Pounce is castable while not Prowling")
	}
	built.ProwlAura.Activate(sim)
	if !built.Pounce.CanCast(sim, target) {
		t.Fatal("Pounce is not castable while Prowling")
	}
}

// A landed Pounce starts the rank's 18 s bleed of six 3 s ticks, awards one
// combo point (effect 30, misc 4, amount 1) and breaks Prowl.
func TestPounceBleedsGrantsComboPointAndBreaksProwl(t *testing.T) {
	built, sim, target := newFeralDruidSimAtLevel(t, 60)

	built.ProwlAura.Activate(sim)
	built.Pounce.ApplyEffects(sim, target, built.Pounce.Spell)

	dot := built.Pounce.Dot(target)
	if !dot.IsActive() {
		t.Fatal("Pounce did not start its bleed")
	}
	if got, want := dot.Aura.Duration, 18*time.Second; got != want {
		t.Errorf("Pounce bleed lasts %v, want %v", got, want)
	}
	if got, want := dot.NumberOfTicks, int32(6); got != want {
		t.Errorf("Pounce bleed has %d ticks, want %d", got, want)
	}
	if got, want := built.ComboPoints(), int32(1); got != want {
		t.Errorf("combo points after a landed Pounce = %d, want %d", got, want)
	}
	if built.ProwlAura.IsActive() {
		t.Fatal("ProwlAura is still active after a landed Pounce")
	}
}
