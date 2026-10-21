package mage

import "testing"

// Forever's talent text (talents/mage.json): Improved Scorch "has a
// 33/67/100% chance".
func TestImprovedScorchChanceIsForevers(t *testing.T) {
	for rank, want := range []float64{0, 0.33, 0.67, 1} {
		if got := improvedScorchProcChance[rank]; got != want {
			t.Errorf("Improved Scorch %d/3 = %v, want %v", rank, got, want)
		}
	}
}
