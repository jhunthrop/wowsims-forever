package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
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
	if got := character.GetStat(stats.MeleeCrit); got != 2*CritRatingPerCritChance {
		t.Errorf("a feral druid's stone gives %v melee crit, want 2", got)
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
