package warrior

import "testing"

// Bloodthrill's per-rank proc chance is the client's trait curve 109366
// on 1289682 e0: 10% at rank 1, 20% at rank 2 (2 ranks since
// 1.60.1.70755; 4% a point over five before). Kept as its own function
// (bloodthrillProcChance) so the ranks are checkable without rolling
// sim.Proc; an over-rank string reads the max rank.
func TestBloodthrillProcChancePerRank(t *testing.T) {
	cases := []struct {
		rank int32
		want float64
	}{
		{0, 0},
		{1, 0.10},
		{2, 0.20},
		{5, 0.20},
	}
	for _, c := range cases {
		if got := bloodthrillProcChance(c.rank); got != c.want {
			t.Errorf("rank %d: proc chance = %v, want %v", c.rank, got, c.want)
		}
	}
	if got, want := len(TalentSpellIDs["bloodthrill"]), len(bloodthrillProcChancePct)-1; got != want {
		t.Errorf("Bloodthrill has %d client ranks, the table %d", got, want)
	}
}
