package paladin

import "testing"

// In 1.60.1.70707 a multi-rank talent is one spell with a trait curve, so
// its buff is one spell too: Vengeance 20049 triggers 20050 at every rank
// (20051 through 20059 have no SpellName row), and Redoubt 20127's buff is
// 20128 (20134 has none). The breakdown names an aura by its id.
func TestTalentBuffsCarryTheClientsIds(t *testing.T) {
	if vengeanceBuffSpellID != 20050 {
		t.Errorf("Vengeance buff id %d, want 20050", vengeanceBuffSpellID)
	}
	if redoubtActionID != 20128 {
		t.Errorf("Redoubt buff id %d, want 20128", redoubtActionID)
	}
}
