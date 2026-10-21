package druid

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestRavageRankAtLevel(t *testing.T) {
	// Level 60 keeps rank 4 (id 9867, learned at 58); source: 1.60.1.70009
	// spellranks.json.
	if got := core.HighestRankAtLevel(ravageLearnLevels, 60); got != 4 {
		t.Errorf("rank at 60 = %d, want 4", got)
	}
	// Between the 42/50 learn levels, level 45 is still rank 2 (id 6787).
	if got := core.HighestRankAtLevel(ravageLearnLevels, 45); got != 2 {
		t.Errorf("rank at 45 = %d, want 2", got)
	}
	// Below Ravage's first learn level (32), nothing is learned yet -- a
	// Feral druid can Prowl from level 20 but has no Ravage until 32.
	if got := core.HighestRankAtLevel(ravageLearnLevels, 31); got != 0 {
		t.Errorf("rank at 31 = %d, want 0", got)
	}
}

// TestRavageFlatDamageBonusMatchesRankTable pins ravageFlatDamageBonus to
// the four numbers verified against 1.60.1.70009's
// spellconst/druid.json (effect 0, base points, spell ids 6785/6787/9866/9867).
func TestRavageFlatDamageBonusMatchesRankTable(t *testing.T) {
	want := [5]float64{0, 42, 62, 78, 98}
	if ravageFlatDamageBonus != want {
		t.Errorf("ravageFlatDamageBonus = %v, want %v", ravageFlatDamageBonus, want)
	}
}
