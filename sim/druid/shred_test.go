package druid

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestShredRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (5, id 9830).
	if got := core.HighestRankAtLevel(shredLearnLevels, 60); got != 5 {
		t.Errorf("rank at 60 = %d, want 5", got)
	}
	// Between the old 25/40 brackets, level 38 is rank 3's own learn level
	// (id 8992) -- the old map returned 0 (a miss) for any level but 25/40/50/60.
	if got := core.HighestRankAtLevel(shredLearnLevels, 38); got != 3 {
		t.Errorf("rank at 38 = %d, want 3", got)
	}
	// Below the first rank's learn level, nothing is learned yet.
	if got := core.HighestRankAtLevel(shredLearnLevels, 21); got != 0 {
		t.Errorf("rank at 21 = %d, want 0", got)
	}
}
