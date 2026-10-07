package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// The Restoration tree's healing talents. Each reads its rank through
// clampRank, as the rest of this package's talents do.
//
// Not modelled for a healer, and why: Nature's Focus (nothing in these fights
// interrupts the druid), Subtlety (threat), Natural Shapeshifter and Furor
// (a healer never shapeshifts; the feral code reads both), and Omen of
// Clarity, which is a trainable passive (spell 16864) rather than a talent
// whose proc rate the client does not publish. Nature's Swiftness, Living
// Spirit, Naturalist's damage half and Reflection are in talents.go.

// giftOfTheEarthmotherGlobalCooldownCut is the half second off the global
// cooldown (spell 414673, "-500" ms on its mask).
const giftOfTheEarthmotherGlobalCooldownCut = 500 * time.Millisecond

const (
	giftOfNatureHealingPerRank      = 2  // percent
	improvedRejuvenationPerRank     = 5  // percent
	tranquilSpiritCostCutPerRank    = 2  // percent
	improvedRegrowthCritPerRank     = 10 // percent
	improvedTranquilityCDCutPerRank = 30 // percent
)

// applyGiftOfNature implements Gift of Nature (node 104916, 5 ranks): "Increases
// the effect of all your healing spells by 2%" a rank. It is a percent modifier
// on the spells (the client's aura 108 on every spell family), so it adds
// to Improved Rejuvenation's, as the two do in the client.
func (druid *Druid) applyGiftOfNature() {
	if druid.Talents.GiftOfNature == 0 {
		return
	}
	rank := clampRank(druid.Talents.GiftOfNature, 5)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_DamageDone_Flat,
		ClassMask: DruidSpellMaskHealing,
		IntValue:  giftOfNatureHealingPerRank * int64(rank),
	})
}

// applyGiftOfTheEarthmother implements Gift of the Earthmother (node 104918,
// bool): "Reduces the global cooldown by 0.5 seconds on your Rejuvenation,
// Swiftmend, and Wild Growth spells."
func (druid *Druid) applyGiftOfTheEarthmother() {
	if !druid.Talents.GiftOfTheEarthmother {
		return
	}
	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_GlobalCooldown_Flat,
		ClassMask: DruidSpellMaskRejuvenation | DruidSpellMaskSwiftmend | DruidSpellMaskWildGrowth,
		TimeValue: -giftOfTheEarthmotherGlobalCooldownCut,
	})
}

// applyTranquilSpirit implements Tranquil Spirit (node 104915, 5 ranks):
// "Reduces the mana cost of your Healing Touch and Tranquility spells by 2%"
// a rank.
func (druid *Druid) applyTranquilSpirit() {
	if druid.Talents.TranquilSpirit == 0 {
		return
	}
	rank := clampRank(druid.Talents.TranquilSpirit, 5)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_PowerCost_Pct,
		ClassMask: DruidSpellMaskHealingTouch | DruidSpellMaskTranquility,
		IntValue:  -tranquilSpiritCostCutPerRank * int64(rank),
	})
}

// applyImprovedRejuvenation implements Improved Rejuvenation (node 104914,
// 3 ranks): "Increases the effect of your Rejuvenation spell by 5%" a rank.
func (druid *Druid) applyImprovedRejuvenation() {
	if druid.Talents.ImprovedRejuvenation == 0 {
		return
	}
	rank := clampRank(druid.Talents.ImprovedRejuvenation, 3)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_DamageDone_Flat,
		ClassMask: DruidSpellMaskRejuvenation,
		IntValue:  improvedRejuvenationPerRank * int64(rank),
	})
}

// applyImprovedTranquility implements Improved Tranquility (node 104909, 2
// ranks): "Reduces threat caused by Tranquility by 50% and its cooldown by
// 30%" at rank 1, 100% and 60% at rank 2. Threat is not modeled (nothing
// in these sims aggroes off a heal), the cooldown is.
func (druid *Druid) applyImprovedTranquility() {
	if druid.Talents.ImprovedTranquility == 0 {
		return
	}
	rank := clampRank(druid.Talents.ImprovedTranquility, 2)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_Cooldown_Multi_Flat,
		ClassMask: DruidSpellMaskTranquility,
		IntValue:  -improvedTranquilityCDCutPerRank * int64(rank),
	})
}

// applyImprovedRegrowth implements Improved Regrowth (node 104913, 5 ranks):
// "Increases the critical effect chance of your Regrowth spell by 10%" a
// rank. Only the direct heal critically strikes, so only it gains.
func (druid *Druid) applyImprovedRegrowth() {
	if druid.Talents.ImprovedRegrowth == 0 {
		return
	}
	rank := clampRank(druid.Talents.ImprovedRegrowth, 5)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Flat,
		ClassMask:  DruidSpellMaskRegrowth,
		FloatValue: improvedRegrowthCritPerRank * float64(rank) * core.CritRatingPerCritChance,
	})
}

// reflectionRanks is Reflection's three ranks (node 104917), each 1/6 of the
// mana regeneration that continues while casting: the client's 17/33/50%.
const reflectionRanks = 3

// applyReflection implements Reflection: "Allows 17/33/50% of your Mana
// regeneration to continue while casting."
func (druid *Druid) applyReflection() {
	rank := clampRank(druid.Talents.Reflection, reflectionRanks)
	druid.PseudoStats.SpiritRegenRateCasting += float64(rank) / reflectionRanks * 0.5
}
