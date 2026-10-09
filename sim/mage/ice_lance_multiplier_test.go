package mage

import "testing"

// TestIceLanceFrozenMultiplierIsFour pins the client's SpellEffect row
// 1342606 (spell 1312002, effect 3, base points 300): "300% increased
// damage" is +300%, so x4 against a Frozen target.
func TestIceLanceFrozenMultiplierIsFour(t *testing.T) {
	if iceLanceFrozenMultiplier != 4.0 {
		t.Errorf("iceLanceFrozenMultiplier = %v, want 4 (base points 300 = +300%%)", iceLanceFrozenMultiplier)
	}
}
