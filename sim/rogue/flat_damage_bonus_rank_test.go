package rogue

import "testing"

// TestSinisterStrikeFlatDamageBonusByRank pins sinisterStrikeFlatDamageBonus
// against spellconst's own effect 121 amount per rank (1.60.1.70009):
// 3, 6, 10, 15, 22, 33, 52, 68. Before this test, ranks 1-3 and 5 carried a
// neighboring rank's number instead of their own.
func TestSinisterStrikeFlatDamageBonusByRank(t *testing.T) {
	want := [9]float64{0, 3, 6, 10, 15, 22, 33, 52, 68}
	if sinisterStrikeFlatDamageBonus != want {
		t.Errorf("sinisterStrikeFlatDamageBonus = %v, want %v", sinisterStrikeFlatDamageBonus, want)
	}
}

// TestBackstabFlatDamageBonusByRank pins backstabFlatDamageBonus against
// spellconst's own effect 121 amount per rank (1.60.1.70009): 10, 20, 32,
// 46, 60, 90, 110, and rank 8's own id (11281 without AQ, 25300 with it)
// carrying 140/150. Before this test, ranks 1, 2, 4 and 7 carried a
// neighboring rank's number instead of their own.
func TestBackstabFlatDamageBonusByRank(t *testing.T) {
	want := [9]float64{0, 10, 20, 32, 46, 60, 90, 110, backstabFlatDamageBonus[8]}
	if backstabFlatDamageBonus != want {
		t.Errorf("backstabFlatDamageBonus = %v, want %v", backstabFlatDamageBonus, want)
	}
	if backstabFlatDamageBonus[8] != 140 && backstabFlatDamageBonus[8] != 150 {
		t.Errorf("backstabFlatDamageBonus[8] = %v, want 140 (no AQ) or 150 (AQ)", backstabFlatDamageBonus[8])
	}
}

// TestAmbushFlatDamageBonusByRank pins ambushFlatDamageBonus against
// spellconst's own effect 121 amount per rank (1.60.1.70009): 28, 40, 50,
// 74, 92, 116. Before this test, ranks 2 and 4 carried the preceding
// rank's number instead of their own.
func TestAmbushFlatDamageBonusByRank(t *testing.T) {
	want := [7]float64{0, 28, 40, 50, 74, 92, 116}
	if ambushFlatDamageBonus != want {
		t.Errorf("ambushFlatDamageBonus = %v, want %v", ambushFlatDamageBonus, want)
	}
}

// TestGarroteBaseDamageByRank pins garroteBaseDamage against spellconst's
// own periodic-damage effect amount per rank (1.60.1.70009): 24, 34, 47,
// 59, 74, 92. Before this test, ranks 1 and 3 carried rank 2's number
// instead of their own.
func TestGarroteBaseDamageByRank(t *testing.T) {
	want := [7]float64{0, 24, 34, 47, 59, 74, 92}
	if garroteBaseDamage != want {
		t.Errorf("garroteBaseDamage = %v, want %v", garroteBaseDamage, want)
	}
}
