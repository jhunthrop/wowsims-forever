package mage

import "testing"

// TestIceLanceFrozenMultiplierIsThree pins the client's SpellEffect row
// 1342606 (spell 1312002, effect 3, base points 200 since 1.60.1.70425):
// "200% increased damage" is +200%, so x3 against a Frozen target.
func TestIceLanceFrozenMultiplierIsThree(t *testing.T) {
	if iceLanceFrozenMultiplier != 3.0 {
		t.Errorf("iceLanceFrozenMultiplier = %v, want 3 (base points 200 = +200%%)", iceLanceFrozenMultiplier)
	}
}
