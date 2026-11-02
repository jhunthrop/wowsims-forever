package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's 30-second potions (1.60.1.70707): Frenzy Potion 250940 (spell
// 1251936, aura 99 and aura 124 of 28), Major Frenzy Potion 250943
// (1251940, 80 and 80), Spellblasting Potion 250934 (1251923, aura 13 of
// 16, school mask 126) and Major Spellblasting Potion 250937 (1251932, 47).
// Each lasts 30 s and sits in the potion category (SpellCategories 4,
// SpellCooldowns 120000 ms), so it shares the two-minute timer with a mana
// or rage potion. Wowhead Forever's tooltips read the same: "Increases
// Attack Power by 80 for 30 sec.", "Increases Spell Damage by 47 for 30 sec."
// The healer's are Mender's Potion 250946 (1251945, aura 135 of 31) and
// Major Mender's Potion 250949 (1251948, 88): "Increases Spell Healing by
// 88 for 30 sec.", in the same category with the same recovery.
func TestBurstPotionsMatchTheClient(t *testing.T) {
	want := map[proto.Potions]burstPotion{
		proto.Potions_FrenzyPotion:             {itemID: 250940, label: "Frenzy Potion", bonus: stats.Stats{stats.AttackPower: 28, stats.RangedAttackPower: 28}},
		proto.Potions_MajorFrenzyPotion:        {itemID: 250943, label: "Major Frenzy Potion", bonus: stats.Stats{stats.AttackPower: 80, stats.RangedAttackPower: 80}},
		proto.Potions_SpellblastingPotion:      {itemID: 250934, label: "Spellblasting Potion", bonus: stats.Stats{stats.SpellDamage: 16}},
		proto.Potions_MajorSpellblastingPotion: {itemID: 250937, label: "Major Spellblasting Potion", bonus: stats.Stats{stats.SpellDamage: 47}},
		proto.Potions_MendersPotion:            {itemID: 250946, label: "Mender's Potion", bonus: stats.Stats{stats.HealingPower: 31}},
		proto.Potions_MajorMendersPotion:       {itemID: 250949, label: "Major Mender's Potion", bonus: stats.Stats{stats.HealingPower: 88}},
	}
	if len(burstPotions) != len(want) {
		t.Errorf("%d burst potions, want %d", len(burstPotions), len(want))
	}
	for potion, expected := range want {
		if got := burstPotions[potion]; got != expected {
			t.Errorf("%v is %+v, want %+v", potion, got, expected)
		}
	}
	if burstPotionDuration != 30*time.Second {
		t.Errorf("burst potions last %v, want 30s", burstPotionDuration)
	}
}

// A burst potion is a DPS cooldown on the shared potion timer: two minutes,
// and its aura gives the potion's stats for 30 s.
func TestBurstPotionIsADPSCooldownOnThePotionTimer(t *testing.T) {
	character := newGnomeEurekaTestCaster(proto.Class_ClassRogue)
	potionCD := character.NewTimer()
	mcd := makePotionActivationInternal(proto.Potions_MajorFrenzyPotion, character, potionCD)
	if mcd.Spell == nil {
		t.Fatal("Major Frenzy Potion has no activation")
	}
	if mcd.Type != CooldownTypeDPS {
		t.Errorf("cooldown type %v, want DPS", mcd.Type)
	}
	if got := mcd.Spell.CD.Duration; got != 2*time.Minute {
		t.Errorf("cooldown %v, want 2m", got)
	}
	if !mcd.SelfBuff {
		t.Error("a burst potion buffs the drinker; a healer with a friendly target must drink it too")
	}
	if mcd.Spell.CD.Timer != potionCD {
		t.Error("Major Frenzy Potion must use the shared potion timer")
	}
	aura := character.GetAura("Major Frenzy Potion")
	if aura == nil {
		t.Fatal("no Major Frenzy Potion aura")
	}
	if aura.Duration != 30*time.Second {
		t.Errorf("aura lasts %v, want 30s", aura.Duration)
	}
}
