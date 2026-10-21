package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

// A "Bent Wand"-shaped item: nonzero ID (0 means "nothing equipped" to
// GetRangedWeapon), a wand-speed-plausible swing speed, and a damage range that
// makes the midpoint easy to check by hand.
func testWand() Item {
	return Item{
		ID:               12345,
		RangedWeaponType: proto.RangedWeaponType_RangedWeaponTypeWand,
		WeaponDamageMin:  6,
		WeaponDamageMax:  12,
		SwingSpeed:       1.6,
	}
}

func TestRegisterShootSpellReturnsNilWithoutAWand(t *testing.T) {
	character := &Character{}

	if spell := character.RegisterShootSpell(1); spell != nil {
		t.Fatalf("RegisterShootSpell registered %v for a character with an empty ranged slot", spell.ActionID)
	}
}

func TestRegisterShootSpellReturnsNilForANonWandRangedWeapon(t *testing.T) {
	character := &Character{}
	character.Equipment[proto.ItemSlot_ItemSlotRanged] = Item{
		ID:               999,
		RangedWeaponType: proto.RangedWeaponType_RangedWeaponTypeBow,
		WeaponDamageMin:  10,
		WeaponDamageMax:  20,
		SwingSpeed:       2.8,
	}

	if spell := character.RegisterShootSpell(1); spell != nil {
		t.Fatalf("RegisterShootSpell registered %v for a bow, not a wand", spell.ActionID)
	}
}

func TestRegisterShootSpellUsesTheEquippedWand(t *testing.T) {
	character := &Character{}
	character.Equipment[proto.ItemSlot_ItemSlotRanged] = testWand()

	const wandDamageMultiplier = 1.25 // e.g. rank-2 Wand Specialization
	spell := character.RegisterShootSpell(wandDamageMultiplier)
	if spell == nil {
		t.Fatal("RegisterShootSpell returned nil for a character with a wand equipped")
	}

	if spell.ActionID.SpellID != ShootSpellID {
		t.Errorf("Shoot ActionID.SpellID = %d, want %d (the client's Shoot spell)", spell.ActionID.SpellID, ShootSpellID)
	}
	if spell.SpellSchool != SpellSchoolPhysical {
		t.Errorf("Shoot SpellSchool = %v, want SpellSchoolPhysical", spell.SpellSchool)
	}
	if spell.DefenseType != DefenseTypeMagic {
		t.Errorf("Shoot DefenseType = %v, want DefenseTypeMagic (wands use spell hit/crit, not weapon skill)", spell.DefenseType)
	}
	if !spell.ProcMask.Matches(ProcMaskRangedAuto) {
		t.Errorf("Shoot ProcMask = %v, want it to match ProcMaskRangedAuto (Blizzard's own ranged-auto-attack proc flag)", spell.ProcMask)
	}
	if spell.Cost != nil {
		t.Errorf("Shoot has a resource cost (%v); wands cost no mana", spell.Cost)
	}
	if spell.DefaultCast.GCD != 0 {
		t.Errorf("Shoot DefaultCast.GCD = %v, want 0 (Shoot does not trigger the global cooldown)", spell.DefaultCast.GCD)
	}

	wantCastTime := time.Duration(1.6 * float64(time.Second))
	if spell.DefaultCast.CastTime != wantCastTime {
		t.Errorf("Shoot DefaultCast.CastTime = %v, want %v (the equipped wand's speed)", spell.DefaultCast.CastTime, wantCastTime)
	}

	if spell.DamageMultiplier != wandDamageMultiplier {
		t.Errorf("Shoot DamageMultiplier = %v, want %v (the caller's wandDamageMultiplier, e.g. Wand Specialization)", spell.DamageMultiplier, wandDamageMultiplier)
	}

	// The base damage Shoot's ApplyEffects rolls from is the equipped wand's own damage
	// range with no attack-power contribution (weapon.CalculateWeaponDamage(sim, 0)), so its
	// deterministic average must equal the wand's own average, independent of any RNG roll.
	wand := character.WeaponFromRanged()
	wantAverage := (testWand().WeaponDamageMin + testWand().WeaponDamageMax) / 2
	if wand.AverageDamage() != wantAverage {
		t.Errorf("equipped wand's AverageDamage() = %v, want %v (the wand's own min/max midpoint)", wand.AverageDamage(), wantAverage)
	}
}

func TestRegisterShootSpellIsIdempotent(t *testing.T) {
	character := &Character{}
	character.Equipment[proto.ItemSlot_ItemSlotRanged] = testWand()

	first := character.RegisterShootSpell(1)
	second := character.RegisterShootSpell(1)

	if first != second {
		t.Fatalf("RegisterShootSpell registered Shoot twice instead of returning the existing spell (GetOrRegisterSpell)")
	}
	if len(character.Spellbook) != 1 {
		t.Fatalf("Spellbook has %d entries after registering Shoot twice, want 1", len(character.Spellbook))
	}
}
