package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// A relic in the ranged slot (a libram, idol or totem the item database
// leaves without a ranged weapon type) is no weapon: a ranged attack by its
// wearer (a paladin's Hammer of Wrath, and since the 11 November 2026
// hotfix every Judgement) rolls on the class's base weapon skill, which
// NewAttackTable adds on its own. GetWeaponSkill returned level x 5 for one
// "for Hammer of Wrath to use 300 skill", doubling the skill: against a
// level 63 boss the ranged table then read a negative miss, no dodge and
// no glance, so Hammer of Wrath never missed below the cap.
func TestARelicInTheRangedSlotAddsNoWeaponSkill(t *testing.T) {
	unit := &Unit{Level: 60}
	relic := &Item{HandType: proto.HandType_HandTypeUnknown, RangedWeaponType: proto.RangedWeaponType_RangedWeaponTypeUnknown}
	if got := GetWeaponSkill(unit, relic); got != 0 {
		t.Errorf("a relic adds %v weapon skill, want 0 (the base skill is the table's own)", got)
	}
	if got := GetWeaponSkill(unit, nil); got != 0 {
		t.Errorf("an empty ranged slot adds %v weapon skill, want 0", got)
	}
}
