package shaman

import (
	"math"
	"testing"
)

// Elemental Weapons (16266) effect 2 is aura 108, misc 3 (effect 0's
// points), +13/27/40 by rank on SpellClassMask_0 0x800000, Windfury
// Weapon's attack power effect (16361: 333 at 60). The text: "your
// Windfury Weapon effect by 40%". So 3/3 makes 333 into 466.2, once.
func TestWindfuryWeaponAttackPowerTakesElementalWeaponsOnce(t *testing.T) {
	if got, want := windfuryWeaponBonusAttackPower(4, 3), 333*1.4; math.Abs(got-want) > 1e-9 {
		t.Errorf("Windfury Weapon rank 4 with Elemental Weapons 3/3 adds %v attack power, want %v", got, want)
	}
	if got, want := windfuryWeaponBonusAttackPower(4, 0), 333.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("Windfury Weapon rank 4 without Elemental Weapons adds %v attack power, want %v", got, want)
	}
}
