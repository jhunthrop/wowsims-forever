package paladin

import (
	"math"
	"testing"
)

// Consecration adds its spell-power share by hand (the first targets' extra
// tick), so it must take Forever's level-gap rule itself: a rank keeps its
// whole coefficient until the caster is 17 levels past the rank's learn
// level, then loses 5% a level (core.LevelPenalty).
func TestConsecrationCoefficientFollowsTheLevelGap(t *testing.T) {
	cases := []struct {
		rankLevel, casterLevel int32
		want                   float64
	}{
		{20, 37, consecrationFirstTargetsCoefficient},
		{20, 40, consecrationFirstTargetsCoefficient * 0.85},
		{20, 50, consecrationFirstTargetsCoefficient * 0.35},
		{20, 60, 0},
		{50, 60, consecrationFirstTargetsCoefficient},
		{60, 60, consecrationFirstTargetsCoefficient},
	}
	for _, c := range cases {
		if got := consecrationCoefficient(c.rankLevel, c.casterLevel); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("rank learned at %d, caster %d: coefficient %v, want %v", c.rankLevel, c.casterLevel, got, c.want)
		}
	}
}
