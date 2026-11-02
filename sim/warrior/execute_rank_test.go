package warrior

import "testing"

// Execute converts each extra point of rage into 3, 6, 9, 12 and 15 damage
// at ranks 1 to 5 (5308, 20658, 20660, 20661, 20662): the client
// description's per-rank figure as Wowhead Forever renders it, the same
// as Era. Every rank used to convert at rank 5's 15.
func TestExecuteRageConversionFollowsTheRank(t *testing.T) {
	want := [ExecuteRanks + 1]float64{0, 3, 6, 9, 12, 15}
	for rank := 1; rank <= ExecuteRanks; rank++ {
		if got := executeRageConversion(rank); got != want[rank] {
			t.Errorf("Execute rank %d converts %v damage per extra rage, want %v", rank, got, want[rank])
		}
	}
}
