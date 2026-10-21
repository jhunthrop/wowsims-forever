package rogue

import "testing"

// Weapon Expertise is a dodge-and-parry reduction in Forever (1% a point,
// two ranks), not weapon skill.
func TestWeaponExpertiseIsOnePercentAPoint(t *testing.T) {
	cases := []struct {
		points int32
		want   float64
	}{{0, 0}, {1, 1}, {2, 2}, {5, 2}}
	for _, c := range cases {
		if got := weaponExpertisePercent(c.points); got != c.want {
			t.Errorf("Weapon Expertise %d = %v, want %v", c.points, got, c.want)
		}
	}
}
