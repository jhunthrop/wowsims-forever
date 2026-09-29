package druid

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestProwlRankAtLevel(t *testing.T) {
	// Level 60 gets rank 3 (id 9913); source: 1.60.1.70009 spellranks.json.
	if got := core.HighestRankAtLevel(prowlLearnLevels, 60); got != 3 {
		t.Errorf("rank at 60 = %d, want 3", got)
	}
	// Between the 20/40 learn levels, level 30 is still rank 1 (id 5215).
	if got := core.HighestRankAtLevel(prowlLearnLevels, 30); got != 1 {
		t.Errorf("rank at 30 = %d, want 1", got)
	}
	// Below Prowl's first learn level (20), nothing is learned yet.
	if got := core.HighestRankAtLevel(prowlLearnLevels, 19); got != 0 {
		t.Errorf("rank at 19 = %d, want 0", got)
	}
}
