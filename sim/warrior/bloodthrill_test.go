package warrior

import "testing"

// Bloodthrill's per-rank proc chance is a flat 4% a point; kept as its
// own function (bloodthrillProcChance) precisely so the five ranks are
// checkable without rolling sim.Proc.
func TestBloodthrillProcChancePerRank(t *testing.T) {
	cases := []struct {
		rank int32
		want float64
	}{
		{1, 0.04},
		{2, 0.08},
		{3, 0.12},
		{4, 0.16},
		{5, 0.20},
	}
	for _, c := range cases {
		if got := bloodthrillProcChance(c.rank); got != c.want {
			t.Errorf("rank %d: proc chance = %v, want %v", c.rank, got, c.want)
		}
	}
}
