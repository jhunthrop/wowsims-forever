package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Forever's Axe Specialization is crit while an axe is equipped in either
// hand (client 20574), so the gate is the equipped weapon type, not weapon
// skill.
func TestHasWeaponOfTypeLooksAtBothHands(t *testing.T) {
	character := &Character{}
	if character.HasWeaponOfType(proto.WeaponType_WeaponTypeSword) {
		t.Fatal("an unarmed character has no sword")
	}
	character.Equipment[proto.ItemSlot_ItemSlotMainHand] = Item{ID: 1, WeaponType: proto.WeaponType_WeaponTypeMace}
	character.Equipment[proto.ItemSlot_ItemSlotOffHand] = Item{ID: 2, WeaponType: proto.WeaponType_WeaponTypeSword}
	if !character.HasWeaponOfType(proto.WeaponType_WeaponTypeMace) || !character.HasWeaponOfType(proto.WeaponType_WeaponTypeSword) {
		t.Fatal("a mace main hand and a sword off hand qualify for both specializations")
	}
	if character.HasWeaponOfType(proto.WeaponType_WeaponTypeAxe) {
		t.Fatal("no axe is equipped")
	}
}

func TestAxeSpecializationCritPercentMatchesTheClient(t *testing.T) {
	if axeSpecializationCritPercent != 1 {
		t.Fatalf("axe = %v, want the client's 1", axeSpecializationCritPercent)
	}
}
