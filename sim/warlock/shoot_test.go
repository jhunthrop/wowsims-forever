package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// core/wand_test.go covers core.RegisterShootSpell itself (school, zero cost, cast time from
// a known wand's speed) and mage/shoot_test.go's TestShootDealsWandDamageAgainstARealTarget
// covers the same path end-to-end against a live target. Warlocks have no Wand
// Specialization equivalent, so this only needs to check registerShootSpell wires the
// package's own Shoot field to what core.RegisterShootSpell(1) returns.
func TestRegisterShootSpellWiresTheShootField(t *testing.T) {
	warlock := &Warlock{}

	warlock.registerShootSpell()
	if warlock.Shoot != nil {
		t.Fatalf("registerShootSpell set Shoot to %v for a warlock with no wand equipped", warlock.Shoot.ActionID)
	}

	warlock.Equipment[proto.ItemSlot_ItemSlotRanged] = core.Item{
		ID:               12345,
		RangedWeaponType: proto.RangedWeaponType_RangedWeaponTypeWand,
		WeaponDamageMin:  6,
		WeaponDamageMax:  12,
		SwingSpeed:       1.6,
	}

	warlock.registerShootSpell()
	if warlock.Shoot == nil {
		t.Fatal("registerShootSpell left Shoot nil for a warlock with a wand equipped")
	}
	if warlock.Shoot.DamageMultiplier != 1 {
		t.Errorf("warlock Shoot DamageMultiplier = %v, want 1 (no Wand Specialization talent to fold in)", warlock.Shoot.DamageMultiplier)
	}
}
