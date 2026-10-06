package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Raging Blows reduces Cleave's rage cost by a flat 2, the same
// SpellMod_PowerCost_Flat shape cost_discounts_test.go already covers
// for the other five discount talents.
func TestRagingBlowsReducesCleaveCost(t *testing.T) {
	without := buildWarriorForCostTest(t, emptyWarriorTalents)
	with := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "raging_blows", 1))

	wantWithout := without.Cleave.Cost.GetCurrentCost()
	if got, want := with.Cleave.Cost.GetCurrentCost(), wantWithout-2; got != want {
		t.Errorf("Cleave costs %v rage with Raging Blows, want %v (undiscounted %v - 2)", got, want, wantWithout)
	}
}

// TestRagingBlowsWhirlwindStrikesOffHand checks the off-hand half:
// Whirlwind must land one extra hit per target when the talent is known
// and the warrior is dual-wielding, and only the main-hand hit
// otherwise - dual-wielding is the real gate, the same one
// weapon_specs_test.go exercises for Dual Wield Specialization via
// weaponWithHandType/buildWarriorWithWeapons.
func TestRagingBlowsWhirlwindStrikesOffHand(t *testing.T) {
	oneHander := weaponWithHandType(t, proto.HandType_HandTypeOneHand)

	withTalent, sim := buildWarriorWithWeaponsAndSim(t, talentStringWithRank(t, emptyWarriorTalents, "raging_blows", 1), oneHander, oneHander)
	target := sim.Encounter.TargetUnits[0]
	withTalent.Whirlwind.ApplyEffects(sim, target, withTalent.Whirlwind.Spell)

	if got := withTalent.Whirlwind.SpellMetrics[target.UnitIndex].Hits; got != 2 {
		t.Errorf("dual-wielding with Raging Blows: Whirlwind landed %d hits on one target, want 2 (MH + OH)", got)
	}

	withoutTalent, sim2 := buildWarriorWithWeaponsAndSim(t, emptyWarriorTalents, oneHander, oneHander)
	target2 := sim2.Encounter.TargetUnits[0]
	withoutTalent.Whirlwind.ApplyEffects(sim2, target2, withoutTalent.Whirlwind.Spell)
	if got := withoutTalent.Whirlwind.SpellMetrics[target2.UnitIndex].Hits; got != 1 {
		t.Errorf("dual-wielding without Raging Blows: Whirlwind landed %d hits on one target, want 1 (MH only)", got)
	}
}

// Without an off-hand weapon, Raging Blows must not add a swing: the
// talent's own clause presumes a weapon to strike with.
func TestRagingBlowsWhirlwindSkipsOffHandWhenNotDualWielding(t *testing.T) {
	twoHander := weaponWithHandType(t, proto.HandType_HandTypeTwoHand)

	war, sim := buildWarriorWithWeaponsAndSim(t, talentStringWithRank(t, emptyWarriorTalents, "raging_blows", 1), twoHander, 0)
	target := sim.Encounter.TargetUnits[0]
	war.Whirlwind.ApplyEffects(sim, target, war.Whirlwind.Spell)

	if got := war.Whirlwind.SpellMetrics[target.UnitIndex].Hits; got != 1 {
		t.Errorf("a 2H warrior with Raging Blows: Whirlwind landed %d hits on one target, want 1 (no off hand to strike with)", got)
	}
}
