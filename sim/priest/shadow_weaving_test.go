package priest

import "testing"

// Shadow Weaving's per-rank chance is the live talent text's 33/67/100%;
// kept as its own function so the three ranks are checkable without
// rolling the sim's random stream.
func TestShadowWeavingProcChancePerRank(t *testing.T) {
	cases := []struct {
		rank int
		want float64
	}{
		{1, 1.0 / 3},
		{2, 2.0 / 3},
		{3, 1},
	}
	for _, c := range cases {
		if got := shadowWeavingProcChance(c.rank); got != c.want {
			t.Errorf("rank %d: proc chance = %v, want %v", c.rank, got, c.want)
		}
	}
}
