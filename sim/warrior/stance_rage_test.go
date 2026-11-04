package warrior

import "testing"

// Tactical Mastery (1310185, trained at 14) keeps 10 rage through a
// stance change for every warrior; Improved Tactical Mastery (12295, 3
// ranks since 1.60.1.70755) adds 5 a point (trait curve 83313: 5/10/15),
// 25 at 3/3.
func TestStanceSwapRetainsBaselineTacticalMastery(t *testing.T) {
	cases := []struct {
		level  int32
		points int32
		want   float64
	}{
		{10, 0, 0},
		{14, 0, 10},
		{60, 0, 10},
		{60, 1, 15},
		{60, 2, 20},
		{60, 3, 25},
	}
	for _, c := range cases {
		if got := retainedStanceRage(c.level, c.points); got != c.want {
			t.Errorf("level %d, %d ITM points: retains %v rage, want %v", c.level, c.points, got, c.want)
		}
	}
}

// The max rank's retained rage is the client row's own: 12295 e0 base
// points 15, over the talent's 3 ranks.
func TestImprovedTacticalMasteryMaxRankIsTheClientRow(t *testing.T) {
	row, ok := clientWarrior(t).ByID(TalentSpellIDs["improved_tactical_mastery"][0])
	if !ok {
		t.Fatal("Improved Tactical Mastery 12295 is not in the client table")
	}
	maxRank := len(TalentSpellIDs["improved_tactical_mastery"])
	if maxRank != 3 {
		t.Errorf("Improved Tactical Mastery has %d ranks, want 3 (1.60.1.70755)", maxRank)
	}
	if got, want := improvedTacticalMasteryRagePerPoint*float64(maxRank), row.Effects[0].Amount; got != want {
		t.Errorf("Improved Tactical Mastery %d/%d adds %v rage, client 12295 e0 %v", maxRank, maxRank, got, want)
	}
}
