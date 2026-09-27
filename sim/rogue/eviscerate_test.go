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
