package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Forever's racial weapon specializations are crit while a weapon of the
// type is equipped in either hand (client 20597, 20574, 1259719), so the
// gate is the equipped weapon type, not weapon skill.
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

func TestWeaponSpecializationCritPercentsMatchTheClient(t *testing.T) {
	if swordSpecializationCritPercent != 2 || axeSpecializationCritPercent != 1 || maceSpecializationCritPercent != 1 {
		t.Fatalf("sword/axe/mace = %v/%v/%v, want the client's 2/1/1", swordSpecializationCritPercent, axeSpecializationCritPercent, maceSpecializationCritPercent)
	}
}
