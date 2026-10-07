package paladin

import "testing"

// Redoubt is 4% block per rank, 20% at rank 5 (the live text and
// Blizzard's 1 October 2026 notes, "Redoubt 4-20%"); the engine read 6%
// a rank, 30% at rank 5, before.
func TestRedoubtBlockPerRank(t *testing.T) {
	if redoubtBlockChancePerRank != 4.0 {
		t.Errorf("redoubtBlockChancePerRank = %v, want 4", redoubtBlockChancePerRank)
	}
	if got := redoubtBlockChancePerRank * 5; got != 20 {
		t.Errorf("rank 5 Redoubt = %v%% block, want 20%%", got)
	}
}
