package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestGarroteRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (6, id 11290).
	if got := core.HighestRankAtLevel(garroteLearnLevels, 60); got != 6 {
		t.Errorf("rank at 60 = %d, want 6", got)
	}
	// Level 38 is rank 4's own learn level (id 8633) -- matches what the old
	// bracket-40 entry gave, but now also correct for every level in between.
	if got := core.HighestRankAtLevel(garroteLearnLevels, 38); got != 4 {
		t.Errorf("rank at 38 = %d, want 4", got)
	}
	if got := core.HighestRankAtLevel(garroteLearnLevels, 13); got != 0 {
		t.Errorf("rank at 13 = %d, want 0", got)
	}
}
