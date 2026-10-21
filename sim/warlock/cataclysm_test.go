package warlock

import "testing"

// Cataclysm, as the live tree states it: "Reduces the Mana cost of your
// Destruction spells by 3% / 6% / 10%".
func TestCataclysmReducesDestructionCostByTheClientsPercent(t *testing.T) {
	_, bare, _ := newBareWarlockForDamageTest(t)
	baseCost := bare.SearingPain[len(bare.SearingPain)-1].Cost.GetCurrentCost()

	for rank, wantPercent := range map[string]float64{"1": 3, "2": 6, "3": 10} {
		// Destruction's fifth talent, after Affliction's 17 and Demonology's 19.
		_, talented, _ := newWarlockForDamageTest(t, "--0000"+rank)
		got := talented.SearingPain[len(talented.SearingPain)-1].Cost.GetCurrentCost()
		want := baseCost * (100 - wantPercent) / 100
		if got < want-1e-6 || got > want+1e-6 {
			t.Errorf("Cataclysm %s: Searing Pain costs %v, want %v (%v%% off %v)", rank, got, want, wantPercent, baseCost)
		}
	}
}

func TestCataclysmLeavesAfflictionSpellsAlone(t *testing.T) {
	_, bare, _ := newBareWarlockForDamageTest(t)
	_, talented, _ := newWarlockForDamageTest(t, "--00003")
	base := bare.Corruption[len(bare.Corruption)-1].Cost.GetCurrentCost()
	if got := talented.Corruption[len(talented.Corruption)-1].Cost.GetCurrentCost(); got != base {
		t.Errorf("Cataclysm changed Corruption's cost from %v to %v", base, got)
	}
}
