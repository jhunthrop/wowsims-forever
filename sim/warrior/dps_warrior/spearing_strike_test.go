package dpswarrior

import "testing"

// Spearing Strike is gated on its talent exactly like the mage's Ice
// Lance: a warrior that did not spend the point must not have the
// spell, or its .results golden could move for an attack it never
// registers.
func TestSpearingStrikeRegistersOnlyWhenTalented(t *testing.T) {
	without := buildWarriorForCostTest(t, emptyWarriorTalents)
	if without.SpearingStrike != nil {
		t.Error("a warrior with no points in Spearing Strike has the spell registered")
	}

	with := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "spearing_strike", 1))
	if with.SpearingStrike == nil {
		t.Fatal("a warrior with Spearing Strike talented has no spell registered")
	}
	if got := with.SpearingStrike.Cost.GetCurrentCost(); got != 15 {
		t.Errorf("Spearing Strike costs %v rage, want 15 (SpellPower.csv row 314497, 150 tenths)", got)
	}
}
