package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestRuptureRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (6, id 11275).
	if got := core.HighestRankAtLevel(ruptureLearnLevels, 60); got != 6 {
		t.Errorf("rank at 60 = %d, want 6", got)
	}
	// Level 38 is rank 3's own learn level (id 8640) -- matches what the old
	// bracket-40 entry gave, but now also correct for every level in between.
	if got := core.HighestRankAtLevel(ruptureLearnLevels, 38); got != 3 {
		t.Errorf("rank at 38 = %d, want 3", got)
	}
	if got := core.HighestRankAtLevel(ruptureLearnLevels, 19); got != 0 {
		t.Errorf("rank at 19 = %d, want 0", got)
	}
}
