package warrior

import (
	"testing"
	"time"
)

// Improved Slam: the cast time and global cooldown each lose 0.25 s a
// point, and the cooldown loses 3 s with any point spent: both rank
// texts read "Slam's cooldown is reduced by 3.0 sec", and client 12862
// (1.60.1.70658) carries a flat -3000 ms on the cooldown, so Slam's 18 s
// becomes 15 s. With any point "Slam no longer interrupts or delays your
// melee swing".
func TestImprovedSlamReductionsPerPoint(t *testing.T) {
	cases := []struct {
		points   int32
		cast     time.Duration
		cooldown time.Duration
	}{
		{0, 0, 0},
		{1, 250 * time.Millisecond, 3 * time.Second},
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
