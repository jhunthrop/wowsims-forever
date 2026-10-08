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

// TestEviscerateComboDamageBonusByRank pins the per-combo-point term
// against the client's EffectPointsPerResource (SpellEffect.csv,
// 1.60.1.70009: 5, 11, 19, 31, 45, 71, 110, 151, 170), which the vendored
// spellconst does not carry. The flat and the width of the roll are
// checked against the client file by TestEviscerateDamageMatchesClient.
// Rank 9 is the AQ rank; without AQ the level-60 slot holds rank 8's term
// (id 11300).
func TestEviscerateComboDamageBonusByRank(t *testing.T) {
	want := [10]float64{0, 5, 11, 19, 31, 45, 71, 110, 151, core.TernaryFloat64(core.IncludeAQ, 170, 151)}
	if eviscerateComboDamageBonus != want {
		t.Errorf("eviscerateComboDamageBonus = %v, want %v", eviscerateComboDamageBonus, want)
	}
}

// Without AQ content the ninth slot casts rank 8's spell 11300, so it takes
// that spell's learn level (56, the client's spell_level for 11300), not the
// 60 of the AQ rank 31016 it would otherwise be filed under.
func TestEviscerateNinthSlotLevelFollowsTheSpellItCasts(t *testing.T) {
	want := core.TernaryInt(core.IncludeAQ, 60, 56)
	if got := eviscerateLearnLevels[len(eviscerateLearnLevels)-1]; got != want {
		t.Errorf("ninth slot learn level = %d, want %d", got, want)
	}
}
