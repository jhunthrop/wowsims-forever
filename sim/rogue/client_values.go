package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Talent and spell numbers read off the client, where the Forever client
// departs from the vanilla figures this engine started with. A talent's
// per-rank value is its trait curve (TraitDefinitionEffectPoints ->
// CurvePoint, build 1.60.1.70755), which is what the client's rank text
// prints; the talent spell's own EffectBasePointsF is not the rank value.
// A Forever patch that changes a number changes a line here and nothing
// else.
const (
	// Deflection: "Increases your Parry chance by 2%/4%/6%." (Vanilla's 1%
	// a rank is not Forever's.)
	deflectionParryPerRank = 2.0
	// Evasion's cooldown. Elusiveness cuts Vanish and Blind, not Evasion
	// ("Reduces the cooldown of your Vanish and Blind abilities by 45/90
	// sec"); Endurance, which cuts Evasion, is not modeled.
	evasionCooldown = 5 * time.Minute
	// Lethality (14128 e0 aura 108 misc 15, crit damage): curve 83046,
	// 10/20% over two ranks.
	lethalityCritBonusPerRank = 0.10
	// Dual Wield Specialization (13715 e0 aura 122): curve 128136, 25/50%
	// over two ranks.
	dualWieldSpecializationPerRank = 0.25
	// Vigor: "Increases your maximum Energy by 5." / "by 10."
	vigorEnergyPerRank = 5.0
	// Serrated Blades: "ignore 3%/6%/9% of your target's Armor".
	serratedBladesArmorPenPctPerRank = 3.0
	// Improved Expose Armor: "Reduces the Energy cost of your Expose Armor
	// ability by 5/10, and refunds 1/2 Combo Points when cast with 5 Combo
	// Points." The client text carries no armor bonus (Classic's +25%/+50%).
	improvedExposeArmorEnergyPerRank = 5.0
	improvedExposeArmorRefundPerRank = 1
	exposeArmorBaseEnergyCost        = 25.0
	fullComboPoints                  = 5
	// Aggression (18427 e0 aura 108, mask 131074 = Sinister Strike 2 +
	// Eviscerate 131072; Backstab's 4 left the mask in 70755): curve
	// 83028, 5/10/15%.
	aggressionDamagePerRank = 0.05

	// Hemorrhage (16511): "100% weapon damage (145% if a Dagger is
	// equipped) and causes the target to take 15% increased Rupture and
	// Garrote damage from the Rogue. Lasts 15 sec." e2 is aura 271 bp 15 on
	// mask 1048832 (Rupture 1048576 + Garrote 256).
	hemorrhageWeaponDamagePct       = 1.00
	hemorrhageDaggerWeaponDamagePct = 1.45
	hemorrhageBleedDamageBonus      = 0.15
	hemorrhageDebuffDuration        = 15 * time.Second
	// Ghostly Strike: "125% (180% if a Dagger is equipped) weapon damage".
	ghostlyStrikeWeaponDamagePct       = 1.25
	ghostlyStrikeDaggerWeaponDamagePct = 1.80
)

// coupDeGraceBonus is Coup de Grace's rank -> additive damage bonus, index
// 0 unused: 14162 e0 aura 108 on mask 131072 (Eviscerate) and e1 aura 108
// on mask_2 4 (Death Mark), both curve 83054/128137: 7/13/20%.
var coupDeGraceBonus = [4]float64{0, 0.07, 0.13, 0.20}

// coupDeGrace is the rogue's Coup de Grace bonus at its talented rank.
func (rogue *Rogue) coupDeGrace() float64 {
	return coupDeGraceBonus[rankIndex(rogue.Talents.CoupDeGrace, coupDeGraceBonus[:])]
}

// improvedExposeArmorRefund is the Combo Points handed back by an Expose
// Armor cast that spent comboPoints.
func improvedExposeArmorRefund(rank, comboPoints int32) int32 {
	if comboPoints < fullComboPoints {
		return 0
	}
	return improvedExposeArmorRefundPerRank * rank
}

// opportunityMultiplier is Opportunity's rank -> damage multiplier for
// Backstab, Garrote, Ambush and Mutilate: 14057 e0 (mask 516 + mask_1 6)
// and e1 (Garrote's periodic, misc 22, mask 256), curves 83021/110341:
// 4/8/12/16/20% over five ranks.
var opportunityMultiplier = [6]float64{1, 1.04, 1.08, 1.12, 1.16, 1.20}

// opportunity is the Opportunity multiplier at the rogue's talented rank.
func (rogue *Rogue) opportunity() float64 {
	return opportunityMultiplier[rankIndex(rogue.Talents.Opportunity, opportunityMultiplier[:])]
}

// aggressionBonus is Aggression's additive damage bonus for Sinister Strike
// and Eviscerate.
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

// Blood Splatter (1329256, one rank): e0 aura 107 misc 1 (duration) on
// Rupture's mask 1048576 and e1 on Garrote's 256. The talent spell's own
// base points are 4000/6000; the trait curves (TraitDefinitionEffectPoints
// 24587/25219, curves 109811/128090) give 6000/9000 ms, which is what the
// client's rank text and Rupture's 70755 tooltip ($<bloodS>) print.
const (
	bloodSplatterRuptureExtra = 6 * time.Second
	bloodSplatterGarroteExtra = 9 * time.Second
)

// registerBloodSplatterAura gives the talent a passive aura, so a rotation
// can ask auraIsKnown(1329256) whether Rupture runs 6 s longer.
func (rogue *Rogue) registerBloodSplatterAura() {
	if !rogue.Talents.BloodSplatter {
		return
	}
	core.MakePermanent(rogue.RegisterAura(core.Aura{
		Label:    "Blood Splatter",
		ActionID: core.ActionID{SpellID: 1329256},
		Duration: core.NeverExpires,
	}))
}

// bloodSplatterTicks is Blood Splatter's extra duration on a bleed, in that
// bleed's ticks: a duration modifier on a periodic aura adds whole ticks.
func (rogue *Rogue) bloodSplatterTicks(extra, tickLength time.Duration) int32 {
	if !rogue.Talents.BloodSplatter {
		return 0
	}
	return int32(extra / tickLength)
}
