package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Forever: "Weapon enchantments and procs now all work while shapeshifted,
// including effects like weightstones, Crusader enchant, and Windfury Totem"
// (the druid deep dive, news.blizzard.com article 24301515). A feral druid's
// Elemental Sharpening Stone (enchant 2506, spell 22755, aura 52) gives it
// its 2% melee crit, and a Windfury Totem reaches it.
func TestWeaponConsumablesReachAShapeshiftedDruid(t *testing.T) {
	character := newGnomeEurekaTestCaster(proto.Class_ClassDruid)
	character.PseudoStats.FeralCombatEnabled = true
	addImbueStats(character, proto.WeaponImbue_ElementalSharpeningStone, true, Cooldown{})
	// The stone is a bonus on the druid's own melee attacks (a form's
	// claw and swing roll the melee table), not a stat a pet could inherit.
	claw := character.RegisterSpell(SpellConfig{
		ActionID:    ActionID{SpellID: 1},
		SpellSchool: SpellSchoolPhysical,
		DefenseType: DefenseTypeMelee,
		ProcMask:    ProcMaskMeleeMHSpecial,
	})
	if got := claw.BonusCritRating; got != 2*CritRatingPerCritChance {
		t.Errorf("a feral druid's stone gives its claw %v crit rating, want 2%%", got)
	}
	if !windfuryTotemReaches(character) {
		t.Error("Windfury Totem must reach a feral druid")
	}
}

// No potion refuses to activate because of a form (the feral and bear
// tests check that drinking one keeps the form).
func TestPotionsDoNotBreakAShapeshift(t *testing.T) {
	for _, potion := range []proto.Potions{
		proto.Potions_MajorManaPotion, proto.Potions_MightyRagePotion,
		proto.Potions_GreaterStoneshieldPotion, proto.Potions_MajorHealingPotion,
		proto.Potions_MajorFrenzyPotion, proto.Potions_MajorSpellblastingPotion,
	} {
		character := newGnomeEurekaTestCaster(proto.Class_ClassDruid)
		mcd := makePotionActivationInternal(potion, character, character.NewTimer())
		if mcd.Spell == nil {
			t.Fatalf("%v has no activation", potion)
		}
		if mcd.ShouldActivate == nil {
			continue
		}
		form := character.RegisterAura(Aura{Label: "Test Form", Duration: NeverExpires})
		character.ActiveShapeShift = nil
		unshifted := activateAnswer(mcd, character)
		character.ActiveShapeShift = form
		if shifted := activateAnswer(mcd, character); shifted != unshifted {
			t.Errorf("%v answers %v in a form and %v out of one", potion, shifted, unshifted)
		}
	}
}

func activateAnswer(mcd MajorCooldown, character *Character) (answer bool) {
	defer func() {
		if recover() != nil {
			answer = false
		}
	}()
	return mcd.ShouldActivate(nil, character)
}

// The flat damage stones and weightstones are weapon enchantments too
// ("including effects like weightstones"), so a feral druid's form attacks
// carry them. The form's auto attack is "the same as the DPS of the
// Druid's equipped weapon", so the stone counts as the weapon's own damage
// per swing at the weapon's speed: a Dense Weightstone's 8 on a 3.2 s staff
// is 8/3.2 damage a second more on the paw.
func TestAFeralDruidsWeightstoneReachesItsWeapon(t *testing.T) {
	for _, c := range []struct {
		imbue proto.WeaponImbue
		bonus float64
	}{
		{proto.WeaponImbue_DenseWeightstone, 8}, {proto.WeaponImbue_SolidWeightstone, 6},
		{proto.WeaponImbue_DenseSharpeningStone, 8}, {proto.WeaponImbue_SolidSharpeningStone, 6},
	} {
		character := newGnomeEurekaTestCaster(proto.Class_ClassDruid)
		character.Equipment[proto.ItemSlot_ItemSlotMainHand] = Item{ID: 1, WeaponDamageMin: 100, WeaponDamageMax: 160, SwingSpeed: 3.2, HandType: proto.HandType_HandTypeTwoHand}
		character.PseudoStats.FeralCombatEnabled = true
		weapon := character.WeaponFromMainHand()
		before := weapon.DPS()
		addImbueStats(character, c.imbue, true, Cooldown{})
		after := character.WeaponFromMainHand()
		if got, want := after.DPS()-before, c.bonus/3.2; got < want-1e-9 || got > want+1e-9 {
			t.Errorf("%v: a feral druid's weapon gains %.4f damage a second, want %.4f", c.imbue, got, want)
		}
	}
}
