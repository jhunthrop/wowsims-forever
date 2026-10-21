package dpswarrior

import (
	"sort"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// shieldItemID mirrors weaponWithHandType (weapon_specs_test.go) for a
// shield: the lowest-numbered item in the loaded database whose weapon
// type is Shield, so Bastion's own test does not pin an id a database
// refresh could retire.
func shieldItemID(t *testing.T) int32 {
	t.Helper()

	var ids []int32
	for id, item := range core.ItemsByID {
		if item.Type == proto.ItemType_ItemTypeWeapon && item.WeaponType == proto.WeaponType_WeaponTypeShield {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		t.Fatal("no shield in the item database; this test needs --tags=with_db")
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids[0]
}

// Improved Revenge is a flat 20% a point on Revenge's own
// DamageMultiplierAdditive - the same SpellMod_DamageDone_Flat shape
// Piercing Ice uses in sim/mage/talents.go - checked here rather than
// named-with-reason despite being a Protection-only talent, because
// unlike its tank-only neighbors it is nothing but a tooltip-line
// percentage on one named spell.
func TestImprovedRevengeAddsDamageMultiplier(t *testing.T) {
	without := buildWarriorForCostTest(t, emptyWarriorTalents)
	for _, points := range []int{1, 2, 3} {
		with := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "improved_revenge", points))
		want := without.Revenge.DamageMultiplierAdditive + 0.20*float64(points)
		if got := with.Revenge.DamageMultiplierAdditive; got != want {
			t.Errorf("%d point(s): Revenge.DamageMultiplierAdditive = %v, want %v", points, got, want)
		}
	}
}

// Focused Rage reduces every SpellFlagOffensive spell's rage cost by a
// flat amount a point - broader than a single ClassMask, which is
// exactly what the client's unqualified "your offensive abilities" asks
// for. Checked against two different offensive specials so the SpellFlags
// filter is shown reaching more than one ClassMask, the way a
// ClassMask-scoped mod could not without naming every mask in a union.
func TestFocusedRageDiscountsEveryOffensiveAbility(t *testing.T) {
	without := buildWarriorForCostTest(t, emptyWarriorTalents)
	with := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "focused_rage", 3))

	if got, want := with.Execute.Cost.GetCurrentCost(), without.Execute.Cost.GetCurrentCost()-3; got != want {
		t.Errorf("Execute costs %v rage with Focused Rage 3, want %v", got, want)
	}
	if got, want := with.Rend.Cost.GetCurrentCost(), without.Rend.Cost.GetCurrentCost()-3; got != want {
		t.Errorf("Rend costs %v rage with Focused Rage 3, want %v", got, want)
	}
}

// Bastion is a flat percentage read off gear state rather than a
// ClassMask, the same shape Two-Handed Weapon Specialization already
// uses above it in talents.go: "all damage" reaches auto-attacks, which
// carry no ClassSpellMask, so it goes on PseudoStats instead of
// becoming a SpellMod, and it is live only while a shield is equipped.
func TestBastionAppliesOnlyWithAShieldEquipped(t *testing.T) {
	shield := shieldItemID(t)
	oneHander := weaponWithHandType(t, proto.HandType_HandTypeOneHand)

	const points = 5
	const want = 1.10

	withShield := buildWarriorWithWeapons(t, talentStringWithRank(t, emptyWarriorTalents, "bastion", points), oneHander, shield)
	withoutShieldBase := buildWarriorWithWeapons(t, emptyWarriorTalents, oneHander, shield)
	if got, base := withShield.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical], withoutShieldBase.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical]; got != base*want {
		t.Errorf("with a shield and %d points the Physical damage multiplier is %v, want %v", points, got, base*want)
	}

	noShield := buildWarriorWithWeapons(t, talentStringWithRank(t, emptyWarriorTalents, "bastion", points), oneHander, oneHander)
	noShieldBase := buildWarriorWithWeapons(t, emptyWarriorTalents, oneHander, oneHander)
	if got, base := noShield.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical], noShieldBase.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical]; got != base {
		t.Errorf("dual-wielding with no shield: the Physical multiplier moved from %v to %v; Bastion must not apply", base, got)
	}
}
