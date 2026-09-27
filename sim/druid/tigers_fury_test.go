package druid

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestTigersFuryRankAtLevel(t *testing.T) {
	// Tiger's Fury is unranked in the current client data: one spell,
	// learned at 24, for every level from 24 through 60.
	if got := core.HighestRankAtLevel(tigersFuryLearnLevels, 60); got != 1 {
		t.Errorf("rank at 60 = %d, want 1", got)
	}
	if got := core.HighestRankAtLevel(tigersFuryLearnLevels, 38); got != 1 {
		t.Errorf("rank at 38 = %d, want 1", got)
	}
	if got := core.HighestRankAtLevel(tigersFuryLearnLevels, 23); got != 0 {
		t.Errorf("rank at 23 = %d, want 0", got)
	}
}
