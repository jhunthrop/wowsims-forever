package core

import "testing"

// Forever: bonus healing carries one third as bonus damage. It is a
// universal dependency, applied to every character, because healing power
// appears on gear for classes that also deal damage.
func TestHealingToSpellDamageRatioIsOneThird(t *testing.T) {
	want := 1.0 / 3.0
	if HealingToSpellDamageRatio != want {
		t.Errorf("HealingToSpellDamageRatio = %v, want %v", HealingToSpellDamageRatio, want)
	}
}
