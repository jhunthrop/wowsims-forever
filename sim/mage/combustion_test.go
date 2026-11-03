package mage

import "testing"

// Combustion ends after three non-periodic Fire critical strikes: the
// client text for build 1.60.1.70009 and Blizzard's 1 October 2026 notes
// ("Combustion 3 charges"; it was 4 in the earlier client tables).
func TestCombustionEndsAfterThreeCriticalStrikes(t *testing.T) {
	if combustionCriticalStrikes != 3 {
		t.Errorf("combustionCriticalStrikes = %d, want 3", combustionCriticalStrikes)
	}
}

// Combustion's buff 28682 stacks to 10 (SpellAuraOptions 135400,
// CumulativeAura 10): +10% Fire crit a stack, so +100% at most.
func TestCombustionStacksToTen(t *testing.T) {
	if combustionMaxStacks != 10 {
		t.Errorf("combustionMaxStacks = %d, want 10", combustionMaxStacks)
	}
}
