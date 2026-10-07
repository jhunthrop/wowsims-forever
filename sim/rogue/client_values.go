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

// opportunityMultiplier is Opportunity's rank -> damage multiplier for
// Backstab, Garrote, Ambush and Mutilate: "by 5%/10%" (two ranks).
var opportunityMultiplier = [3]float64{1, 1.05, 1.10}

// mainHandStrikePct picks a main-hand strike's weapon-damage fraction: the
// dagger figure with a dagger in the main hand, the plain one otherwise.
func (rogue *Rogue) mainHandStrikePct(plain, withDagger float64) float64 {
	if rogue.HasDagger(core.MainHand) {
		return withDagger
	}
	return plain
}
