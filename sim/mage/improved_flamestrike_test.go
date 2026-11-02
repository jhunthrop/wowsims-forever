package mage

import (
	"github.com/wowsims/classic/sim/core"
	"testing"
)

// Improved Flamestrike (11108) is aura 107 misc 7, 15 points at 3/3 on
// Flamestrike's mask. The spell's config also added 5 a rank on top of the
// declarative mod, so Flamestrike had 30% over the Fire crit every Fire
// spell shares (Critical Mass, read off Fireball here).
func TestImprovedFlamestrikeIsFifteenPercentAtThreeRanks(t *testing.T) {
	talents := talentStringWithRank(t, round10FireRaidTalents, "improved_flamestrike", 3)
	built, _, _ := newFrostMageSimWithTalents(t, bossTarget(), talents)
	flamestrike := topRank(t, built.Flamestrike)
	fireball := topRank(t, built.Fireball)
	if got, want := flamestrike.BonusCritRating-fireball.BonusCritRating, float64(15*core.CritRatingPerCritChance); got != want {
		t.Errorf("Improved Flamestrike 3/3 adds %.1f crit rating to Flamestrike, want %.1f", got, want)
	}
}
