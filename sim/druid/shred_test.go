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
}

// TestShredRankOneIsLearnedAtTwenty: client 1.60.1.70707 has Shred rank 1
// (5221) at level 20 (SpellLevels 98857 BaseLevel 20, SkillLineAbility on
// Feral Combat), Wowhead Forever 5221 "Requires level 20", and Blizzard's
// 15 October beta notes: "Rank 1 Shred is learned at level 20 instead of 22."
func TestShredRankOneIsLearnedAtTwenty(t *testing.T) {
	if got := core.HighestRankAtLevel(shredLearnLevels, 20); got != 1 {
		t.Errorf("rank at 20 = %d, want 1", got)
	}
	if got := core.HighestRankAtLevel(shredLearnLevels, 19); got != 0 {
		t.Errorf("rank at 19 = %d, want 0", got)
	}
}
