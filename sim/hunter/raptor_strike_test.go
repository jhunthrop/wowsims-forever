package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestRaptorStrikeRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (8, id 14266).
	if got := core.HighestRankAtLevel(RaptorStrikeLevel[1:], 60); got != 8 {
		t.Errorf("rank at 60 = %d, want 8", got)
	}
	// Between the old 32/40 brackets, level 38 is still rank 5 (learned at
	// 32); the old map returned 0 (a miss) for any level but 25/40/50/60.
	if got := core.HighestRankAtLevel(RaptorStrikeLevel[1:], 38); got != 5 {
		t.Errorf("rank at 38 = %d, want 5", got)
	}
	if got := core.HighestRankAtLevel(RaptorStrikeLevel[1:], 0); got != 0 {
		t.Errorf("rank at 0 = %d, want 0", got)
	}
}
