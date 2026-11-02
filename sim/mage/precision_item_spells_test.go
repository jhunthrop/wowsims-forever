package mage

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
	"math"
	"testing"
)

// Precision (29438) is aura 55, spell hit chance, with no class mask:
// "Improves your chance to hit with all spells by 6%". A spell an item
// casts for the mage (Kindling Stave's Ignition, a trinket's bolt) carries
// no mage class mask and must still get the 6%.
func TestPrecisionReachesSpellsWithoutAMageClassMask(t *testing.T) {
	built, _, target := newFrostMageSimWithTalents(t, bossTarget(), round10FrostRaidTalents)
	itemSpell := &core.Spell{Unit: &built.Unit, SpellSchool: core.SpellSchoolFire, SchoolIndex: stats.SchoolIndexFire}
	gearHit := built.GetStat(stats.Hit) / (core.HitRatingPerHitChance * 100)
	if got, want := itemSpell.SpellHitChance(target)-gearHit, 0.06; math.Abs(got-want) > 1e-9 {
		t.Errorf("an item's Fire spell gets %.4f hit from Precision 3/3, want %.2f", got, want)
	}
	bolt := topRank(t, built.Frostbolt)
	if got, want := bolt.SpellHitChance(target)-gearHit, 0.06; math.Abs(got-want) > 1e-9 {
		t.Errorf("Frostbolt gets %.4f hit from Precision 3/3, want %.2f (once, not twice)", got, want)
	}
}
