package mage

import "testing"

// Forever's 1.60.1.70291 talent text (talents/mage.json): Improved Scorch
// "has a 33/67/100% chance", Winter's Chill "Stacks up to 1/2/3/4/5 times".
func TestImprovedScorchChanceIsForevers(t *testing.T) {
	for rank, want := range []float64{0, 0.33, 0.67, 1} {
		if got := improvedScorchProcChance[rank]; got != want {
			t.Errorf("Improved Scorch %d/3 = %v, want %v", rank, got, want)
		}
	}
}

func TestWintersChillStacksOnePerPoint(t *testing.T) {
	for rank := int32(0); rank <= 5; rank++ {
		if got := wintersChillMaxStacks(rank); got != rank {
			t.Errorf("Winter's Chill %d/5 stacks to %d, want %d", rank, got, rank)
		}
	}
	if got := wintersChillMaxStacks(9); got != 5 {
		t.Errorf("an out-of-range rank stacks to %d, want the aura's 5", got)
	}
}
