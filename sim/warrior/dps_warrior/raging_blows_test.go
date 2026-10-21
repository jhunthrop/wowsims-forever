package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Raging Blows reduces the rage cost of Cleave and Whirlwind by a flat 3,
// the same SpellMod_PowerCost_Flat shape cost_discounts_test.go already
// covers for the other discount talents. The client's text says Cleave
// by 2 and names no Whirlwind discount; Blizzard's 1 October 2026 notes
// ("Raging Blows now reduces the rage cost of Cleave and Whirlwind by 3")
// are the live state.
func TestRagingBlowsReducesCleaveAndWhirlwindCost(t *testing.T) {
	without := buildWarriorForCostTest(t, emptyWarriorTalents)
	with := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "raging_blows", 1))

	if got, want := with.Cleave.Cost.GetCurrentCost(), without.Cleave.Cost.GetCurrentCost()-3; got != want {
		t.Errorf("Cleave costs %v rage with Raging Blows, want %v (undiscounted - 3)", got, want)
	}
	if got, want := with.Whirlwind.Cost.GetCurrentCost(), without.Whirlwind.Cost.GetCurrentCost()-3; got != want {
		t.Errorf("Whirlwind costs %v rage with Raging Blows, want %v (undiscounted - 3)", got, want)
	}
}

// TestWhirlwindStrikesOffHandBaseline: Blizzard's 1 October 2026 notes
// make Whirlwind strike with both weapons baseline, so a dual-wielder
// lands one extra hit per target with or without Raging Blows, and the
// dual-wielding gate is the only thing that decides it - the same gate
// weapon_specs_test.go exercises for Dual Wield Specialization via
// weaponWithHandType/buildWarriorWithWeapons.
func TestWhirlwindStrikesOffHandBaseline(t *testing.T) {
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
	if got := withoutTalent.Whirlwind.SpellMetrics[target2.UnitIndex].Hits; got != 2 {
		t.Errorf("dual-wielding without Raging Blows: Whirlwind landed %d hits on one target, want 2 (MH + OH, baseline)", got)
	}
}

// Without an off-hand weapon, Whirlwind must not add a swing: the
// clause presumes a weapon to strike with.
func TestWhirlwindSkipsOffHandWhenNotDualWielding(t *testing.T) {
	twoHander := weaponWithHandType(t, proto.HandType_HandTypeTwoHand)

	war, sim := buildWarriorWithWeaponsAndSim(t, talentStringWithRank(t, emptyWarriorTalents, "raging_blows", 1), twoHander, 0)
	target := sim.Encounter.TargetUnits[0]
	war.Whirlwind.ApplyEffects(sim, target, war.Whirlwind.Spell)

	if got := war.Whirlwind.SpellMetrics[target.UnitIndex].Hits; got != 1 {
		t.Errorf("a 2H warrior Whirlwind landed %d hits on one target, want 1 (no off hand to strike with)", got)
	}
}
