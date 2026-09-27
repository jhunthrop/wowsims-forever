package druid

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestSwipeBearRankAtLevel(t *testing.T) {
	// The old SoD-bracket map gave rank 6 at level 60, which does not exist
	// (SwipeSpellId/SwipeBaseDamage/SwipeLevel only go to rank 5) -- that
	// bracket would have panicked with an out-of-range index had this dead
	// code ever been wired up. The real rank at 60 is 5 (id 9908).
	if got := core.HighestRankAtLevel(SwipeLevel[1:], 60); got != 5 {
		t.Errorf("rank at 60 = %d, want 5", got)
	}
	// Between the old 34/44 brackets, level 38 is still rank 3 (learned at 34).
	if got := core.HighestRankAtLevel(SwipeLevel[1:], 38); got != 3 {
		t.Errorf("rank at 38 = %d, want 3", got)
	}
	if got := core.HighestRankAtLevel(SwipeLevel[1:], 15); got != 0 {
		t.Errorf("rank at 15 = %d, want 0", got)
	}
}
