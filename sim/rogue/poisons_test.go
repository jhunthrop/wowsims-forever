package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestInstantPoisonRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (6, id 11340).
	if got := core.HighestRankAtLevel(instantPoisonLearnLevels, 60); got != 6 {
		t.Errorf("rank at 60 = %d, want 6", got)
	}
	// Between the old 36/44 brackets, level 38 is still rank 3 (learned at
	// 36); the old map returned 0 (a miss) for any level but 25/40/50/60.
	if got := core.HighestRankAtLevel(instantPoisonLearnLevels, 38); got != 3 {
		t.Errorf("rank at 38 = %d, want 3", got)
	}
	if got := core.HighestRankAtLevel(instantPoisonLearnLevels, 19); got != 0 {
		t.Errorf("rank at 19 = %d, want 0", got)
	}
}

func TestDeadlyPoisonRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (5; the id is
	// 25347 with IncludeAQ and 11356 without).
	if got := core.HighestRankAtLevel(deadlyPoisonLearnLevels, 60); got != 5 {
		t.Errorf("rank at 60 = %d, want 5", got)
	}
	// The old bracket map registered Deadly Poison at level 25, before its
	// real learn level of 30 -- level 25 must now correctly resolve to no
	// rank learned at all.
	if got := core.HighestRankAtLevel(deadlyPoisonLearnLevels, 25); got != 0 {
		t.Errorf("rank at 25 = %d, want 0 (old bracket map wrongly registered a rank here)", got)
	}
	// Level 42 is still rank 2 (learned at 38); the old map returned 0 (a
	// miss) for any level but 25/40/50/60.
	if got := core.HighestRankAtLevel(deadlyPoisonLearnLevels, 42); got != 2 {
		t.Errorf("rank at 42 = %d, want 2", got)
	}
}
