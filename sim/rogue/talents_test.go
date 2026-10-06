package rogue

import "testing"

// TestRankIndexClampsAnOverRankToTheTablesMaxEntry mirrors
// mage/talents_test.go's test of the same name: core.FillTalentsProto
// does not validate a talent string against the client's per-node max
// rank, so a corrupt or hand-edited string could otherwise index
// setupComboPointChance out of range instead of reading Setup's
// max-rank (3) value.
func TestRankIndexClampsAnOverRankToTheTablesMaxEntry(t *testing.T) {
	if got, want := setupComboPointChance[rankIndex(9, setupComboPointChance[:])], setupComboPointChance[len(setupComboPointChance)-1]; got != want {
		t.Errorf("setupComboPointChance at rank 9 = %v, want the max-rank value %v", got, want)
	}
	// A rank inside the table still reads its own entry, not the max.
	if got, want := setupComboPointChance[rankIndex(2, setupComboPointChance[:])], setupComboPointChance[2]; got != want {
		t.Errorf("setupComboPointChance at rank 2 = %v, want %v", got, want)
	}
	// A negative rank (shouldn't occur, but rankIndex's own bounds check
	// covers it) must not panic either.
	if got, want := setupComboPointChance[rankIndex(-1, setupComboPointChance[:])], setupComboPointChance[len(setupComboPointChance)-1]; got != want {
		t.Errorf("setupComboPointChance at rank -1 = %v, want the max-rank value %v", got, want)
	}
}

// TestSetupComboPointChanceTableValues pins the client's own 33%/67%/
// 100% rank text (data/builds/1.60.1.70009/talents/rogue.json's Setup
// node) - not an exact 33.33%/rank multiply, so this is a table lookup
// rather than a formula, the same shape mage/talents.go uses for its own
// non-linear per-rank values.
func TestSetupComboPointChanceTableValues(t *testing.T) {
	want := [4]float64{0, 0.33, 0.67, 1.0}
	if setupComboPointChance != want {
		t.Errorf("setupComboPointChance = %v, want %v", setupComboPointChance, want)
	}
}
