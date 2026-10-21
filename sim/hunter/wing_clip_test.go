package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestWingClipRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (3, id 14268).
	if got := core.HighestRankAtLevel(wingClipLearnLevels, 60); got != 3 {
		t.Errorf("rank at 60 = %d, want 3", got)
	}
	// The old bracket map disagreed with the client here: it gave rank 3 at
	// level 50, but rank 3 (id 14268) is not learned until 60 -- the correct
	// rank at 50 is 2 (id 14267, learned at 38).
	if got := core.HighestRankAtLevel(wingClipLearnLevels, 50); got != 2 {
		t.Errorf("rank at 50 = %d, want 2 (old bracket map wrongly gave 3)", got)
	}
	if got := core.HighestRankAtLevel(wingClipLearnLevels, 11); got != 0 {
		t.Errorf("rank at 11 = %d, want 0", got)
	}
}
