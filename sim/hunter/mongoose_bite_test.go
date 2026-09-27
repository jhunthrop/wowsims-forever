package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestMongooseBiteRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (4, id 14271).
	if got := core.HighestRankAtLevel(mongooseBiteLearnLevels, 60); got != 4 {
		t.Errorf("rank at 60 = %d, want 4", got)
	}
	// Between the old 30/44 brackets, level 38 is still rank 2 (learned at 30);
	// the old map returned 0 (a miss) for any level but 25/40/50/60.
	if got := core.HighestRankAtLevel(mongooseBiteLearnLevels, 38); got != 2 {
		t.Errorf("rank at 38 = %d, want 2", got)
	}
	if got := core.HighestRankAtLevel(mongooseBiteLearnLevels, 15); got != 0 {
		t.Errorf("rank at 15 = %d, want 0", got)
	}
}
