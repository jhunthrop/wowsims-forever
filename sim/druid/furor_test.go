package druid

import (
	"testing"
	"time"
)

// TestFurorCatFormEnergyFollowsTheLiveText pins the Cat Form half of Furor
// (node 104958, build 1.60.1.70009): "you will regain 20% of the Energy you
// had when you were last in Cat Form, plus 2 Energy for each second you spent
// not in Bear Form, Cat Form, or Dire Bear Form, up to a maximum of 20 Energy"
// per rank (40/60/80/100 at ranks 2-5).
func TestFurorCatFormEnergyFollowsTheLiveText(t *testing.T) {
	cases := []struct {
		name          string
		rank          int32
		lastCatEnergy float64
		outOfForm     time.Duration
		want          float64
	}{
		{"no talent regains nothing", 0, 80, 5 * time.Second, 0},
		{"rank 1 keeps a fifth of the old energy", 1, 10, 0, 2},
		{"rank 5 keeps all of it", 5, 30, 0, 30},
		{"rank 3 keeps 60 percent", 3, 50, 0, 30},
		{"time out of form adds 2 a second a rank", 2, 0, 3 * time.Second, 12},
		{"fraction and time add", 5, 20, 2 * time.Second, 40},
		{"rank 1 is capped at 20", 1, 100, 30 * time.Second, 20},
		{"rank 5 is capped at 100", 5, 100, 30 * time.Second, 100},
	}
	for _, c := range cases {
		if got := furorCatFormEnergy(c.rank, c.lastCatEnergy, c.outOfForm); got != c.want {
			t.Errorf("%s: furorCatFormEnergy(%d, %v, %v) = %v, want %v", c.name, c.rank, c.lastCatEnergy, c.outOfForm, got, c.want)
		}
	}
}
