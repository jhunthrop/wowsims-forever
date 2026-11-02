package mage

import (
	"github.com/wowsims/classic/sim/core"
	"testing"
)

// The published level-60 raid Fire build (data/builds/1.60.1.70707/bis/
// mage-fire.json): Critical Mass 3/3, Incineration 5/5, Ignite 5/5 and
// Combustion.
const round10FireRaidTalents = "253005000000200000-51521001230133031-003000000000000000"

// Combustion's buff (client 28682) is aura 107 misc 7 on the class mask
// 0x00C00017 / 0x1000: Fireball, Scorch, Pyroblast, Fire Blast,
// Flamestrike, Blast Wave and Frostfire Bolt. Forever's Ignite burn (412538)
// carries the mask 0x08000000, outside it, so Combustion's crit must not
// reach the burn's ticks.
func TestCombustionCritDoesNotReachIgnite(t *testing.T) {
	built, sim, _ := newFrostMageSimWithTalents(t, bossTarget(), round10FireRaidTalents)
	if built.CombustionAura == nil || built.igniteTick == nil {
		t.Fatal("Combustion and Ignite are both expected with the raid Fire build")
	}
	scorch := topRank(t, built.Scorch)
	scorchBefore, igniteBefore := scorch.BonusCritRating, built.igniteTick.BonusCritRating

	built.CombustionAura.Activate(sim)
	built.CombustionAura.SetStacks(sim, 3)

	if got, want := scorch.BonusCritRating-scorchBefore, float64(30*core.CritRatingPerCritChance); got != want {
		t.Errorf("three Combustion stacks add %.1f crit rating to Scorch, want %.1f", got, want)
	}
	if got := built.igniteTick.BonusCritRating - igniteBefore; got != 0 {
		t.Errorf("three Combustion stacks add %.1f crit rating to Ignite's burn, want 0 (412538 is outside 28682's mask)", got)
	}
}
