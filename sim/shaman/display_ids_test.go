package shaman

import "testing"

// Flurry 16256 is one talent spell with a trait curve in 1.60.1.70707; its
// buff is 16257 at every rank (16277 through 16280 have no SpellName row).
func TestFlurryBuffCarriesTheClientsId(t *testing.T) {
	for points := int32(1); points <= 5; points++ {
		if got := flurryBuffSpellID(points); got != 16257 {
			t.Errorf("Flurry %d/5 buff id %d, want 16257", points, got)
		}
	}
}

// Elemental Devastation 30160 is one talent spell with a trait curve
// (3/6/9%) in 1.60.1.70707; its buff is 30165 at every rank (29177 and
// 29178 have no SpellName row).
func TestElementalDevastationBuffCarriesTheClientsId(t *testing.T) {
	for points := int32(1); points <= 3; points++ {
		if got := elementalDevastationBuffSpellID(points); got != 30165 {
			t.Errorf("Elemental Devastation %d/3 buff id %d, want 30165", points, got)
		}
	}
}
