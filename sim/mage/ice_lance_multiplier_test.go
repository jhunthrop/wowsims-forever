package mage

import "testing"

// TestIceLanceFrozenMultiplierIsFour pins client 1.60.1.70707's talent
// text: "Deals 300% increased damage to Frozen targets" is +300%, so x4
// against a Frozen target (x3 from 70425 through 70685).
func TestIceLanceFrozenMultiplierIsFour(t *testing.T) {
	if iceLanceFrozenMultiplier != 4.0 {
		t.Errorf("iceLanceFrozenMultiplier = %v, want 4 (+300%%)", iceLanceFrozenMultiplier)
	}
}
