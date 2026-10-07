package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestEviscerateRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (9, since
	// IncludeAQ is true in this build).
	if got := core.HighestRankAtLevel(eviscerateLearnLevels, 60); got != 9 {
		t.Errorf("rank at 60 = %d, want 9", got)
	}
	// Between the old 32/40 brackets, level 36 is rank 5 (learned at 32);
	// the old map returned 0 (a miss) for any level but 25/40/50/60.
	if got := core.HighestRankAtLevel(eviscerateLearnLevels, 36); got != 5 {
		t.Errorf("rank at 36 = %d, want 5", got)
	}
}

// TestEviscerateDamageTermsByRank pins the flat, range and per-combo-point
// terms against the client (SpellEffect.csv, 1.60.1.70009). The roll is
// centred on EffectBasePointsF (3, 7, 13, 20, 30, 44, 68, 96, 108) with a
// total width of base * Variance (1.333, 1.143, 1.077, then 1 for ranks 4-9),
// so the flat term is base - width/2 and the range is width; the
// per-combo-point term is EffectPointsPerResource (5, 11, 19, 31, 45, 71,
// 110, 151, 170). Rank 9 is the AQ rank; without AQ the level-60 slot holds
// rank 8's numbers (id 11300).
func TestEviscerateDamageTermsByRank(t *testing.T) {
	wantFlat := [10]float64{0, 1, 3, 6, 10, 15, 22, 34, 48, core.TernaryFloat64(core.IncludeAQ, 54, 48)}
	wantRange := [10]float64{0, 4, 8, 14, 20, 30, 44, 68, 96, core.TernaryFloat64(core.IncludeAQ, 108, 96)}
	wantPerComboPoint := [10]float64{0, 5, 11, 19, 31, 45, 71, 110, 151, core.TernaryFloat64(core.IncludeAQ, 170, 151)}
	if eviscerateFlatDamage != wantFlat {
		t.Errorf("eviscerateFlatDamage = %v, want %v", eviscerateFlatDamage, wantFlat)
	}
	if eviscerateDamageVariance != wantRange {
		t.Errorf("eviscerateDamageVariance = %v, want %v", eviscerateDamageVariance, wantRange)
	}
	if eviscerateComboDamageBonus != wantPerComboPoint {
		t.Errorf("eviscerateComboDamageBonus = %v, want %v", eviscerateComboDamageBonus, wantPerComboPoint)
	}
}
