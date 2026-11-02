package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// Round 9 (physical lane): Forever's Expose Armor takes Sunder Armor's
// numbers, per combo point: 90, 180, 270, 360 and 450 armor at ranks 1 to 5
// (8647, 8649, 8650, 11197, 11198, learned at 14, 26, 36, 46, 56), so a
// five-point rank 5 is 2250 (Wowhead Forever tooltips, read 2026-11-02;
// the client rows carry aura 674 at 0, a server-computed amount). Era's
// rank 5 was 340 a point, 1700. Forever's Improved Expose Armor (14168)
// cuts the energy cost by 10 and refunds 2 combo points at five; it no
// longer raises the armor removed, so the raid toggle's "improved" setting
// removes the same armor as the regular one (a five-point Expose Armor of
// the raid's rank).
func TestExposeArmorIsForevers(t *testing.T) {
	checkRanks(t, "Expose Armor", ExposeArmorRanks, []levelWant{{13, 0}, {14, 90}, {26, 180}, {36, 270}, {46, 360}, {55, 360}, {56, 450}, {60, 450}})
	for _, c := range []struct {
		level int32
		armor float64
	}{{60, 2250}, {40, 1350}} {
		target := newDebuffTarget(c.level)
		aura := ExposeArmorAura(target)
		aura.Activate(&Simulation{})
		if got := target.GetStat(stats.Armor); got != -c.armor {
			t.Errorf("level %d: armor after the raid's Expose Armor = %v, want %v", c.level, got, -c.armor)
		}
	}
}
