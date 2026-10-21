package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestSliceAndDiceRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (2, id 6774).
	if got := core.HighestRankAtLevel(sliceAndDiceLearnLevels, 60); got != 2 {
		t.Errorf("rank at 60 = %d, want 2", got)
	}
	// Between the old 25/40 brackets, level 38 is still rank 1 (learned at
	// 10) -- the old map already covered 25 and 40 correctly, but any other
	// level (e.g. 38) was a miss before this fix.
	if got := core.HighestRankAtLevel(sliceAndDiceLearnLevels, 38); got != 1 {
		t.Errorf("rank at 38 = %d, want 1", got)
	}
	if got := core.HighestRankAtLevel(sliceAndDiceLearnLevels, 9); got != 0 {
		t.Errorf("rank at 9 = %d, want 0", got)
	}
}
