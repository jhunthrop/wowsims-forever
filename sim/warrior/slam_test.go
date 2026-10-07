package warrior

import (
	"testing"
	"time"
)

// Improved Slam, per the client's own rank text (build 1.60.1.70009,
// spell 12862): the cast time and global cooldown each lose 0.25 s a
// point, the cooldown loses 1.5 s a point, and with any point spent
// "Slam no longer interrupts or delays your melee swing". Before
// 2026-10-07 the engine took 0.1 s a point off the cast time only and
// stopped the swing for every Slam.
func TestImprovedSlamReductionsPerPoint(t *testing.T) {
	cases := []struct {
		points   int32
		cast     time.Duration
		cooldown time.Duration
	}{
		{0, 0, 0},
		{1, 250 * time.Millisecond, 1500 * time.Millisecond},
		{2, 500 * time.Millisecond, 3 * time.Second},
	}
	for _, c := range cases {
		cast, gcd, cooldown := improvedSlamReductions(c.points)
		if cast != c.cast || gcd != c.cast || cooldown != c.cooldown {
			t.Errorf("points %d: reductions = (cast %s, gcd %s, cooldown %s), want (%s, %s, %s)",
				c.points, cast, gcd, cooldown, c.cast, c.cast, c.cooldown)
		}
	}
}

func TestImprovedSlamKeepsTheSwingWithAnyPoint(t *testing.T) {
	if improvedSlamKeepsTheSwing(0) {
		t.Error("untalented Slam must still stop the swing timer for its cast")
	}
	for _, points := range []int32{1, 2} {
		if !improvedSlamKeepsTheSwing(points) {
			t.Errorf("%d point(s): Slam must no longer delay the melee swing", points)
		}
	}
}
