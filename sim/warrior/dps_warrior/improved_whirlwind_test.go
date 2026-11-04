package dpswarrior

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
)

// Improved Whirlwind (Fury node 105977, which held Raging Blows until
// 1.60.1.70755): "Reduces the cooldown of your Whirlwind ability by 1/2
// sec" (1310315 e0, aura 107 misc 11, trait curve -1000/-2000 ms), so the
// client's 10 s Whirlwind is 9 s and 8 s on a built warrior.
func TestImprovedWhirlwindCutsWhirlwindsCooldown(t *testing.T) {
	for points, want := range map[int]time.Duration{0: 10 * time.Second, 1: 9 * time.Second, 2: 8 * time.Second} {
		war := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "improved_whirlwind", points))
		if got := war.Whirlwind.CD.Duration; got != want {
			t.Errorf("Improved Whirlwind %d/2: Whirlwind's cooldown is %s, want %s", points, got, want)
		}
	}
}

// Raging Blows' rage discount left with Raging Blows: the node's new
// talent touches neither Cleave's nor Whirlwind's cost.
func TestImprovedWhirlwindDiscountsNoRage(t *testing.T) {
	with := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "improved_whirlwind", 2))
	if got := with.Whirlwind.Cost.GetCurrentCost(); got != 25 {
		t.Errorf("Whirlwind costs %v rage with Improved Whirlwind, want the client's 25", got)
	}
	if got := with.Cleave.Cost.GetCurrentCost(); got != 20 {
		t.Errorf("Cleave costs %v rage with Improved Whirlwind, want the client's 20", got)
	}
}

// "...increases the damage it deals to the closest enemy by 15%/30%"
// (1310315 e1, trait curve 15/30): against the one target, the cast's
// own and closest, the same seeded swing deals 1.3 times the damage at
// 2/2, on both the main hand and the off-hand strike.
func TestImprovedWhirlwindRaisesTheClosestEnemysDamage(t *testing.T) {
	oneHander := weaponWithHandType(t, proto.HandType_HandTypeOneHand)
	damage := func(points int) (float64, float64) {
		war, sim := buildWarriorWithWeaponsAndSim(t, talentStringWithRank(t, emptyWarriorTalents, "improved_whirlwind", points), oneHander, oneHander)
		target := sim.Encounter.TargetUnits[0]
		war.Whirlwind.ApplyEffects(sim, target, war.Whirlwind.Spell)
		offHand := war.GetSpell(core.ActionID{SpellID: warrior.WhirlwindOffHandSpellID})
		return war.Whirlwind.SpellMetrics[target.UnitIndex].TotalDamage, offHand.SpellMetrics[target.UnitIndex].TotalDamage
	}
	baseMH, baseOH := damage(0)
	talentMH, talentOH := damage(2)
	if baseMH == 0 || baseOH == 0 {
		t.Fatalf("the seeded Whirlwind dealt no damage (MH %v, OH %v); pick a seed that lands both", baseMH, baseOH)
	}
	for hand, got := range map[string]float64{"main hand": talentMH / baseMH, "off hand": talentOH / baseOH} {
		if math.Abs(got-1.3) > 1e-9 {
			t.Errorf("Improved Whirlwind 2/2: the closest enemy takes %v times the %s's damage, want 1.3", got, hand)
		}
	}
}

// TestWhirlwindStrikesOffHandBaseline: Blizzard's 1 October 2026 notes
// make Whirlwind strike with both weapons baseline, so a dual-wielder
// lands one extra hit per target, and the dual-wielding gate is the only
// thing that decides it - the same gate weapon_specs_test.go exercises
// for Dual Wield Specialization via weaponWithHandType/buildWarriorWithWeapons.
func TestWhirlwindStrikesOffHandBaseline(t *testing.T) {
	oneHander := weaponWithHandType(t, proto.HandType_HandTypeOneHand)

	war, sim := buildWarriorWithWeaponsAndSim(t, emptyWarriorTalents, oneHander, oneHander)
	target := sim.Encounter.TargetUnits[0]
	war.Whirlwind.ApplyEffects(sim, target, war.Whirlwind.Spell)
	if got := outcomes(war.Whirlwind.Spell) + outcomes(war.GetSpell(core.ActionID{SpellID: warrior.WhirlwindOffHandSpellID})); got != 2 {
		t.Errorf("dual-wielding: Whirlwind rolled %d outcomes on one target across 1680 and its off-hand strike, want 2 (MH + OH, baseline)", got)
	}
}

// Without an off-hand weapon, Whirlwind must not add a swing: the
// clause presumes a weapon to strike with.
func TestWhirlwindSkipsOffHandWhenNotDualWielding(t *testing.T) {
	twoHander := weaponWithHandType(t, proto.HandType_HandTypeTwoHand)

	war, sim := buildWarriorWithWeaponsAndSim(t, emptyWarriorTalents, twoHander, 0)
	target := sim.Encounter.TargetUnits[0]
	war.Whirlwind.ApplyEffects(sim, target, war.Whirlwind.Spell)

	if got := war.Whirlwind.SpellMetrics[target.UnitIndex].Hits; got != 1 {
		t.Errorf("a 2H warrior Whirlwind landed %d hits on one target, want 1 (no off hand to strike with)", got)
	}
}
