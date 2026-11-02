package druid

import (
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
// Spirit, Naturalist's damage half and Reflection are in talents.go, Bloom
// in bloom.go. Heart of the Wild (Restoration tier 0 since client
// 1.60.1.70707) is in feral_talents.go: its Intellect reads for a healer.

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
//
// Tranquility is left out. Effect 0 (the DAMAGE op) names it, but Forever
// gives a periodic heal only effect 1 (the DOT op), which names
// Rejuvenation, Regrowth and Wild Growth. In the beta logs (foreverlogs.gg
// reports 2748 and 2770, read 2026-11-02) two level-35 druids' Tranquility
// ticks fit their Healing Touch's healing power only without Gift of Nature.
func (druid *Druid) applyGiftOfNature() {
	if druid.Talents.GiftOfNature == 0 {
		return
	}
	rank := clampRank(druid.Talents.GiftOfNature, 5)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_DamageDone_Flat,
		ClassMask: DruidSpellMaskHealing &^ DruidSpellMaskTranquility,
		IntValue:  giftOfNatureHealingPerRank * int64(rank),
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
// rank. The spell's crit chance reaches its heal over time's ticks too,
// which crit since client 1.60.1.70707.
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

// reflectionRanks is Reflection's three ranks (node 104920), each 1/6 of the
// mana regeneration that continues while casting: the client's 17/33/50%.
const reflectionRanks = 3

// applyReflection implements Reflection: "Allows 17/33/50% of your Mana
// regeneration to continue while casting."
func (druid *Druid) applyReflection() {
	rank := clampRank(druid.Talents.Reflection, reflectionRanks)
	druid.PseudoStats.SpiritRegenRateCasting += float64(rank) / reflectionRanks * 0.5
}
