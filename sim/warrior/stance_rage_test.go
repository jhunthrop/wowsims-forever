package warrior

import "testing"

// Tactical Mastery (1310185, trained at 14) keeps 10 rage through a
// stance change for every warrior; Improved Tactical Mastery (12295) adds
// 3 a point, 25 at 5/5.
func TestStanceSwapRetainsBaselineTacticalMastery(t *testing.T) {
	cases := []struct {
		level  int32
		points int32
		want   float64
	}{
		{10, 0, 0},
		{14, 0, 10},
		{60, 0, 10},
		{60, 5, 25},
	}
	for _, c := range cases {
		if got := retainedStanceRage(c.level, c.points); got != c.want {
			t.Errorf("level %d, %d ITM points: retains %v rage, want %v", c.level, c.points, got, c.want)
		}
	}
}
