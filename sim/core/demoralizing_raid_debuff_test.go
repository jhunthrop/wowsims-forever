package core

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// The raid's Demoralizing Shout and Demoralizing Roar options are the
// client's ranks at the raid's level (spellconst warrior.json 1160, 6190,
// 11554, 11555, 11556 and druid.json 99, 1735, 9490, 9747, 9898: aura 99,
// base points plus points per level to the rank's max level). Rank 5 at 60
// is 196 + 6 x 1.4 = 204.4 for the shout and 193 + 8 x 1.4 = 204.2 for the
// roar, 204 each as whole points. The engine gave Era's 146 and 138 (x1.4 and x1.4 "improved");
// Forever has no Improved Demoralizing Shout or Roar talent, so the
// improved option is the same debuff.
func TestRaidDemoralizeIsTheClients(t *testing.T) {
	checkRanks(t, "Demoralizing Shout", DemoralizingShoutRaidRanks, []levelWant{
		{13, 0}, {14, 49}, {20, 57}, {40, 106}, {54, 196}, {60, 204},
	})
	checkRanks(t, "Demoralizing Roar", DemoralizingRoarRaidRanks, []levelWant{
		{9, 0}, {10, 46}, {20, 74}, {30, 88}, {40, 103}, {60, 204},
	})
	for _, c := range []struct {
		name string
		aura func(*Unit) *Aura
		ap   float64
	}{
		{"Demoralizing Shout", func(u *Unit) *Aura { return DemoralizingShoutAura(u) }, 204},
		{"Demoralizing Roar", func(u *Unit) *Aura { return DemoralizingRoarAura(u) }, 204},
	} {
		target := newDebuffTarget(60)
		c.aura(target).Activate(&Simulation{})
		if got := target.GetStat(stats.AttackPower); math.Abs(got+c.ap) > 1e-9 {
			t.Errorf("%s at 60: attack power %v, want %v", c.name, got, -c.ap)
		}
	}
}
