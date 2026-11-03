package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Mutilate needs a dagger in each hand: every Mutilate rank's
// SpellEquippedItems row is item class 2 with subclass mask 32768, the
// dagger bit (Backstab carries the same row).
func TestMutilateNeedsDaggersInBothHands(t *testing.T) {
	for _, c := range []struct {
		name string
		gear *proto.EquipmentSpec
		want bool
	}{{"two daggers", testDaggers, true}, {"a mace and a sword", testMaceAndSword, false}} {
		sim, rogue := buildRogueWithGear(t, rogueTalentStringWith(t, "mutilate"), core.FullBuffs.Raid, core.FullBuffs.Debuffs, c.gear)
		if got := rogue.Mutilate.CanCast(sim, sim.Encounter.TargetUnits[0]); got != c.want {
			t.Errorf("%s: Mutilate castable = %v, want %v", c.name, got, c.want)
		}
	}
}

// Seal Fate fires off Mutilate: a hand that crits adds one combo point,
// once per cast. With Seal Fate 5 (100%) and every hit forced to crit, a
// Mutilate grants its 2 points plus exactly 1.
func TestSealFateProcsOnceOffAMutilateCrit(t *testing.T) {
	sim, rogue := buildRogueWithGear(t, sealFateFive(t), core.FullBuffs.Raid, core.FullBuffs.Debuffs, testDaggers)
	rogue.MutilateMH.BonusCritRating = 200 * core.CritRatingPerCritChance
	rogue.MutilateOH.BonusCritRating = 200 * core.CritRatingPerCritChance
	rogue.MutilateMH.BonusHitRating = 100 * core.HitRatingPerHitChance
	rogue.MutilateOH.BonusHitRating = 100 * core.HitRatingPerHitChance
	target := sim.Encounter.TargetUnits[0]
	rogue.Mutilate.ApplyEffects(sim, target, rogue.Mutilate)
	if got := rogue.ComboPoints(); got != 3 {
		t.Errorf("combo points after a double-crit Mutilate with Seal Fate 5 = %d, want 3", got)
	}
}

// sealFateFive is Mutilate plus Seal Fate at its 5 ranks.
func sealFateFive(t *testing.T) string {
	t.Helper()
	talents := []rune(rogueTalentStringWith(t, "mutilate", "seal_fate"))
	for i, r := range talents {
		if r == '1' && i == sealFateIndex(t) {
			talents[i] = '5'
		}
	}
	return string(talents)
}

// sealFateIndex is Seal Fate's position in the dash-joined talent string.
func sealFateIndex(t *testing.T) int {
	t.Helper()
	only := rogueTalentStringWith(t, "seal_fate")
	for i, r := range only {
		if r == '1' {
			return i
		}
	}
	t.Fatal("Seal Fate has no position in the talent string")
	return -1
}
