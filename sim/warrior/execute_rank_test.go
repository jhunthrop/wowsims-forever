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

// A percent cost cut on Execute (Gnome's Eureka!, the only one that reaches
// it) lowers what Execute takes up front but converts no extra rage: in
// forever-bugs #439's attached log (Zeyf, Gnome, level 31, 16 October) an
// Execute at 105 rage under Eureka! hit for 434, the unbuffed 395 (125 plus
// 90 rage at 3) times 1.1, not 439.45, which 1.5 more converted rage would
// give. Flat cuts (Improved Execute) still convert, as in Classic.
func TestExecuteConvertsNoRageFromAPercentCostCut(t *testing.T) {
	cases := []struct {
		name          string
		rageLeft      float64
		paid          float64
		multiplierPct int32
		want          float64
	}{
		{"no cut", 90, 15, 100, 90},
		{"Eureka! at 105 rage", 91.5, 13.5, 90, 90},
		{"Eureka! with Improved Execute 2", 96, 9, 90, 95},
		{"Eureka! with too little rage left", 0.5, 13.5, 90, 0},
	}
	for _, c := range cases {
		if got := executeConvertedRage(c.rageLeft, c.paid, c.multiplierPct); got != c.want {
			t.Errorf("%s: converts %v rage, want %v", c.name, got, c.want)
		}
	}
}
