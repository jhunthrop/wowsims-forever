package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestExposeArmorRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (5, id 11198).
	if got := core.HighestRankAtLevel(exposeArmorLearnLevels, 60); got != 5 {
		t.Errorf("rank at 60 = %d, want 5", got)
	}
	// Level 38 is rank 3's own learn level (id 8650) -- matches what the old
	// bracket-40 entry gave, but now also correct for every level in between.
	if got := core.HighestRankAtLevel(exposeArmorLearnLevels, 38); got != 3 {
		t.Errorf("rank at 38 = %d, want 3", got)
	}
	if got := core.HighestRankAtLevel(exposeArmorLearnLevels, 13); got != 0 {
		t.Errorf("rank at 13 = %d, want 0", got)
	}
}
