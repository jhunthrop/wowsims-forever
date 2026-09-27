package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestSinisterStrikeRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (8, id 11294).
	if got := core.HighestRankAtLevel(sinisterStrikeLearnLevels, 60); got != 8 {
		t.Errorf("rank at 60 = %d, want 8", got)
	}
	// Level 38 is rank 6's own learn level (id 8621) -- matches what the old
	// bracket-40 entry gave, but now also correct for every level in between.
	if got := core.HighestRankAtLevel(sinisterStrikeLearnLevels, 38); got != 6 {
		t.Errorf("rank at 38 = %d, want 6", got)
	}
}
