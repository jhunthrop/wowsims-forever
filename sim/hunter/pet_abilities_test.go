package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestPetClawRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (8, id 3009).
	if got := core.HighestRankAtLevel(petClawLearnLevels, 60); got != 8 {
		t.Errorf("rank at 60 = %d, want 8", got)
	}
	// Between the old 32/40 brackets, level 38 is still rank 5 (learned at
	// 32); the old map returned 0 (a miss) for any level but 25/40/50/60.
	if got := core.HighestRankAtLevel(petClawLearnLevels, 38); got != 5 {
		t.Errorf("rank at 38 = %d, want 5", got)
	}
}

func TestPetBiteRankAtLevel(t *testing.T) {
	if got := core.HighestRankAtLevel(petBiteLearnLevels, 60); got != 8 {
		t.Errorf("rank at 60 = %d, want 8", got)
	}
	if got := core.HighestRankAtLevel(petBiteLearnLevels, 38); got != 5 {
		t.Errorf("rank at 38 = %d, want 5", got)
	}
}

func TestPetLightningBreathRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (6, id 25012).
	if got := core.HighestRankAtLevel(petLightningBreathLearnLevels, 60); got != 6 {
		t.Errorf("rank at 60 = %d, want 6", got)
	}
	// Level 38 is rank 4 (learned at 36); the old code deliberately reused
	// rank 3's id here for a "not available in SoD Phase 2" reason that no
	// longer applies now that every phase is released.
	if got := core.HighestRankAtLevel(petLightningBreathLearnLevels, 38); got != 4 {
		t.Errorf("rank at 38 = %d, want 4", got)
	}
}

func TestPetScorpidPoisonRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (4, id 24587).
	if got := core.HighestRankAtLevel(petScorpidPoisonLearnLevels, 60); got != 4 {
		t.Errorf("rank at 60 = %d, want 4", got)
	}
	// Between the old 24/40 brackets, level 38 is still rank 2 (learned at
	// 24); the old map returned 0 (a miss) for any level but 25/40/50/60.
	if got := core.HighestRankAtLevel(petScorpidPoisonLearnLevels, 38); got != 2 {
		t.Errorf("rank at 38 = %d, want 2", got)
	}
}
