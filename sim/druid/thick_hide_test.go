package druid

import (
	"math"
	"testing"
)

// Thick Hide's armor per level is the client's 1 / 2 / 3 by rank plus 3 a
// level above level 1 at every rank (16929 effect 1, curve 111990 and
// EffectRealPointsPerLevel 3 in 1.60.1.70790): 180 at level 60 on the top
// rank, as before, and 178 on the first, not 60. The defense half stays
// 0.67 / 1.33 / 2.00 a point.
func TestThickHideArmorPerLevelIsTheSameAtEveryRank(t *testing.T) {
	for _, c := range []struct {
		rank, level int32
		defense     float64
		want        float64
	}{
		{1, 16, 0, 46}, {2, 16, 0, 47}, {3, 16, 0, 48},
		{1, 60, 0, 178}, {3, 60, 0, 180},
		{3, 60, 30, 240}, {1, 60, 30, 198},
		{0, 60, 30, 0},
	} {
		if got := thickHideBaseArmor(c.rank, c.level, c.defense); math.Abs(got-c.want) > 1e-6 {
			t.Errorf("Thick Hide %d/3 at level %d with %v defense = %v, want %v", c.rank, c.level, c.defense, got, c.want)
		}
	}
}
