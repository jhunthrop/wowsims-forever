package item_effects

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// The ranged-hit items' client restriction (SpellEquippedItems subclass
// mask 262156) accepts bows, guns, crossbows and thrown weapons: a relic,
// a wand or an empty slot gives the wearer nothing.
func TestWieldsRangedWeapon(t *testing.T) {
	cases := []struct {
		name string
		kind proto.RangedWeaponType
		want bool
	}{
		{"empty slot", proto.RangedWeaponType_RangedWeaponTypeUnknown, false},
		{"libram", proto.RangedWeaponType_RangedWeaponTypeLibram, false},
		{"wand", proto.RangedWeaponType_RangedWeaponTypeWand, false},
		{"bow", proto.RangedWeaponType_RangedWeaponTypeBow, true},
		{"gun", proto.RangedWeaponType_RangedWeaponTypeGun, true},
		{"thrown", proto.RangedWeaponType_RangedWeaponTypeThrown, true},
	}
	for _, tc := range cases {
		character := &core.Character{}
		character.Equipment[proto.ItemSlot_ItemSlotRanged] = core.Item{RangedWeaponType: tc.kind}
		if got := wieldsRangedWeapon(character); got != tc.want {
			t.Errorf("%s: wieldsRangedWeapon = %v, want %v", tc.name, got, tc.want)
		}
	}
}
