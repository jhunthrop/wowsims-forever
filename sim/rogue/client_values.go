package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Talent and spell numbers read off the client's own text for build
// 1.60.1.70009 (data/builds/1.60.1.70009/talents/rogue.json and
// spellconst/rogue.json), where the Forever client departs from the
// vanilla figures this engine started with. A Forever patch that changes a
// number changes a line here and nothing else.
const (
	// Lethality: "...critical strike damage bonus ... by 4%/8%/12%/16%/20%."
	lethalityCritBonusPerRank = 0.04
	// Dual Wield Specialization: "off-hand weapon by 5%/10%/15%/20%/25%."
	dualWieldSpecializationPerRank = 0.05
	// Vigor: "Increases your maximum Energy by 5." / "by 10."
	vigorEnergyPerRank = 5.0
	// Murder: "Increases all damage dealt by 2%/4% against Humanoid and
	// Giant targets."
	murderDamagePerRank = 0.02
	// Serrated Blades: "ignore 3%/6%/9% of your target's Armor".
	serratedBladesArmorPenPctPerRank = 3.0
	// Improved Expose Armor: "Reduces the Energy cost of your Expose Armor
	// ability by 5/10, and refunds 1/2 Combo Points when cast with 5 Combo
	// Points." The client text carries no armor bonus (Classic's +25%/+50%).
	improvedExposeArmorEnergyPerRank = 5.0
	improvedExposeArmorRefundPerRank = 1
	exposeArmorBaseEnergyCost        = 25.0
	fullComboPoints                  = 5
	// Aggression: "Increases the damage of your Sinister Strike, Backstab,
	// and Eviscerate abilities by 2%/4%/6%."
	aggressionDamagePerRank = 0.02

	// Hemorrhage (16511): "100% weapon damage (145% if a Dagger is
	// equipped) and causes the target to take 15% increased Rupture damage
	// from the Rogue. Lasts 15 sec."
	hemorrhageWeaponDamagePct       = 1.00
	hemorrhageDaggerWeaponDamagePct = 1.45
	hemorrhageRuptureDamageBonus    = 0.15
	hemorrhageDebuffDuration        = 15 * time.Second
	// Ghostly Strike: "125% (180% if a Dagger is equipped) weapon damage".
	ghostlyStrikeWeaponDamagePct       = 1.25
	ghostlyStrikeDaggerWeaponDamagePct = 1.80
)

// improvedEviscerateMultiplier is Improved Eviscerate's rank -> damage
// multiplier, index 0 unused: "by 7%/13%/20%".
var improvedEviscerateMultiplier = [4]float64{1, 1.07, 1.13, 1.20}

// improvedExposeArmorRefund is the Combo Points handed back by an Expose
// Armor cast that spent comboPoints.
func improvedExposeArmorRefund(rank, comboPoints int32) int32 {
	if comboPoints < fullComboPoints {
		return 0
	}
	return improvedExposeArmorRefundPerRank * rank
}

// opportunityMultiplier is Opportunity's rank -> damage multiplier for
// Backstab, Garrote, Ambush and Mutilate: "by 5%/10%" (two ranks).
var opportunityMultiplier = [3]float64{1, 1.05, 1.10}

// aggressionBonus is Aggression's additive damage bonus for Sinister Strike,
// Backstab and Eviscerate.
func (rogue *Rogue) aggressionBonus() float64 {
	return aggressionDamagePerRank * float64(rogue.Talents.Aggression)
}

// mainHandStrikePct picks a main-hand strike's weapon-damage fraction: the
// dagger figure with a dagger in the main hand, the plain one otherwise.
func (rogue *Rogue) mainHandStrikePct(plain, withDagger float64) float64 {
	if rogue.HasDagger(core.MainHand) {
		return withDagger
	}
	return plain
}
