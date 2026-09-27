package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestAmbushRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (6, id 11269).
	if got := core.HighestRankAtLevel(ambushLearnLevels, 60); got != 6 {
		t.Errorf("rank at 60 = %d, want 6", got)
	}
	// Between the old 34/42 brackets, level 38 is still rank 3 (learned at
	// 34, id 8725) -- the old map returned 0 (a miss) for any level but
	// 25/40/50/60.
	if got := core.HighestRankAtLevel(ambushLearnLevels, 38); got != 3 {
		t.Errorf("rank at 38 = %d, want 3", got)
	}
	if got := core.HighestRankAtLevel(ambushLearnLevels, 17); got != 0 {
		t.Errorf("rank at 17 = %d, want 0", got)
	}
}
