package core

import "testing"

// Thistle Tea is drunk when its 100 energy fits under the cap with at most
// 10 lost. A 100-energy bar drinks at 10 or less, as before; a 110-energy
// bar (Vigor, a gnome's Expansive Mind) drinks at 20 or less. A fixed "10
// or less" left a gnome Assassination rogue, whose bar is 110 and seldom
// bottoms out, without a single tea in a fight (sim-audit round 18).
func TestThistleTeaFitsUnderTheBarsOwnCap(t *testing.T) {
	for _, c := range []struct {
		current, max float64
		want         bool
	}{
		{10, 100, true},
		{11, 100, false},
		{20, 110.25, true},
		{21, 110.25, false},
		{15, 105, true},
		{0, 100, true},
	} {
		if got := thistleTeaFits(c.current, c.max); got != c.want {
			t.Errorf("thistleTeaFits(%v, %v) = %v, want %v", c.current, c.max, got, c.want)
		}
	}
}
