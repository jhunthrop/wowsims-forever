package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestBackstabRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (8).
	if got := core.HighestRankAtLevel(backstabLearnLevels, 60); got != 8 {
		t.Errorf("rank at 60 = %d, want 8", got)
	}
	// Between the old 20/28 brackets, level 22 is still rank 3 (learned at
	// 20, id 2590) -- the old map returned 0 (a miss) for any level but
	// 25/40/50/60.
	if got := core.HighestRankAtLevel(backstabLearnLevels, 22); got != 3 {
		t.Errorf("rank at 22 = %d, want 3", got)
	}
	if got := core.HighestRankAtLevel(backstabLearnLevels, 3); got != 0 {
		t.Errorf("rank at 3 = %d, want 0", got)
	}
}
