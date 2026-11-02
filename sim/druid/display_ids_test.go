package druid

import "testing"

// Furor is one talent spell, 17056, at every rank in 1.60.1.70707 (17058
// through 17061 have no SpellName row).
func TestFurorCarriesTheClientsId(t *testing.T) {
	if furorSpellID != 17056 {
		t.Errorf("Furor id %d, want 17056", furorSpellID)
	}
}
