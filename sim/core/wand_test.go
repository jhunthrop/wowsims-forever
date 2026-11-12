package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
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
	// Shoot (5019) flies at the client's SpellMisc Speed, 20 (the engine had 24).
	if spell.MissileSpeed != 20 {
		t.Errorf("Shoot MissileSpeed = %v, want the client's 20", spell.MissileSpeed)
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

// Torch of Light (279246) states "91 - 170 Arcane Damage" (client ItemSparse
// DamageType 6; Wowhead Forever tooltip). A wand hit is magic of that school,
// so armor never reduces it and spell damage never adds to it.
func TestShootDealsTheWandsOwnSchool(t *testing.T) {
	item := ItemFromProto(&proto.SimItem{
		Id:               279246,
		RangedWeaponType: proto.RangedWeaponType_RangedWeaponTypeWand,
		WeaponDamageMin:  91,
		WeaponDamageMax:  170,
		WeaponSpeed:      1.7,
		DamageSchool:     proto.SpellSchool_SpellSchoolArcane,
	})
	character := &Character{}
	character.Equipment[proto.ItemSlot_ItemSlotRanged] = item

	spell := character.RegisterShootSpell(1)
	if spell == nil {
		t.Fatal("RegisterShootSpell returned nil for a wand")
	}
	if spell.SpellSchool != SpellSchoolArcane {
		t.Errorf("Shoot SpellSchool = %v, want SpellSchoolArcane (the wand's own damage school)", spell.SpellSchool)
	}
	if spell.SchoolIndex == stats.SchoolIndexPhysical {
		t.Errorf("Shoot reads the physical school index, so armor would reduce it")
	}
	if spell.BonusCoefficient != 0 {
		t.Errorf("Shoot BonusCoefficient = %v, want 0 (spell damage does not add to a wand)", spell.BonusCoefficient)
	}
}

func TestShootWithNoStatedSchoolStaysPhysical(t *testing.T) {
	character := &Character{}
	character.Equipment[proto.ItemSlot_ItemSlotRanged] = testWand()

	spell := character.RegisterShootSpell(1)
	if spell.SpellSchool != SpellSchoolPhysical {
		t.Errorf("Shoot SpellSchool = %v, want SpellSchoolPhysical when the item states no school", spell.SpellSchool)
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

// A wand's first shot of a run fires after a 0.5 s windup measured from the
// end of the caster's last spell, not after a full wand speed, and a shot
// inside a run waits only for the wand's swing timer. Read from foreverlogs
// reports 2820, 2839 and 2847 (2026-11-12 sim-accuracy, shadow): every shot
// is logged as a start and a fire 0.50 s apart; after an instant the first
// start sits 1.0 to 1.3 s after the cast (the fire at the global cooldown's
// end); after a channel the first fire sits 0.2 to 1.1 s after its end; and
// shots in a run are the wand's speed apart.
func TestShootCastTimeIsTheWindupLeftOrTheSwingLeft(t *testing.T) {
	const swing = 1600 * time.Millisecond
	const long = -time.Hour
	cases := []struct {
		name                        string
		now, lastSpellEnd, lastShot time.Duration
		want                        time.Duration
	}{
		{"first shot at the end of an instant's global cooldown", 1500 * time.Millisecond, 0, long, 0},
		{"first shot right after a hardcast or channel ends", 3 * time.Second, 3 * time.Second, long, wandWindup},
		{"first shot 200 ms after a cast ended", 3200 * time.Millisecond, 3 * time.Second, long, 300 * time.Millisecond},
		{"a shot inside a run waits the swing", 5 * time.Second, long, 5 * time.Second, swing},
		{"a shot inside a run with 1 s of the swing left", 5600 * time.Millisecond, long, 5 * time.Second, 1000 * time.Millisecond},
		{"the longer of windup and swing wins", 5 * time.Second, 5 * time.Second, 4800 * time.Millisecond, swing - 200*time.Millisecond},
		{"nothing ever cast", 0, long, long, 0},
	}
	for _, c := range cases {
		if got := shootCastTime(c.now, c.lastSpellEnd, c.lastShot, swing); got != c.want {
			t.Errorf("%s: shootCastTime = %v, want %v", c.name, got, c.want)
		}
	}
}
