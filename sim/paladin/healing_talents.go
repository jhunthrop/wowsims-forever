package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Holy tree talents that change a heal (talents/paladin.json, build
// 1.60.1.70009). Every number is the client's own rank text.
const (
	// Healing Light (node 20237): "Increases the amount healed by your Holy
	// Light, Flash of Light, and Holy Shock spells by 4%" a rank.
	healingLightPctPerRank = 0.04

	// Spiritual Focus (node 20205, 5 ranks): "a 14% chance to not lose
	// casting time when you take damage" a rank, 70% at rank 5 (client
	// 1.60.1.70707; 35% a rank over 2 ranks before). The engine's name for
	// that chance is the spell's pushback reduction.
	spiritualFocusPushbackReductionPerRank = 0.14

	// Infusion of Light (node 426065): a Holy Shock or Flash of Light crit
	// "reduce[s] the cast time of your next Holy Light cast within 15 sec
	// by 0.5 sec" a rank.
	infusionOfLightActionID        = 426065
	infusionOfLightDuration        = 15 * time.Second
	infusionOfLightCastTimePerRank = 500 * time.Millisecond

	// Illumination (node 20210): a 20% chance a rank, on a heal crit, to
	// gain 50% of the spell's base mana cost.
	illuminationActionID            = 20210
	illuminationRegenSpellID        = 20272
	illuminationRegenWhileCasting   = 0.50
	illuminationRegenDuration       = 15 * time.Second
	illuminationChancePerRank       = 0.20
	illuminationManaShareOfBaseCost = 0.50

	// Holy Power (node 5923, 5 ranks): "Increases the critical strike
	// healing bonus of your Holy spells by 10%" a rank, 50% at rank 5
	// (client 1.60.1.70707; it raised crit chance before). The client's
	// crit-bonus modifier scales the bonus, as Ruin's does: a heal crit's
	// 50% bonus becomes 75% at rank 5. The paladin's Holy heals are the
	// spells Illumination names.
	holyPowerCritBonusPerRank = 0.10

	// Boundless Light (node 113631, 2 ranks): "Increases the critical
	// strike chance of your Holy Shock spell by 15% when cast on allies
	// not affected by your Oath of Light", 30% at rank 2. Oath of Light is
	// unmodeled (talents.go), so no ally carries it and every Holy Shock
	// heal gets the bonus.
	boundlessLightCritPctPerRank = 15.0
)

// applyHealingTalents is the healing-relevant part of the Holy tree. Divine
// Favor and Light's Vigil are spells and live in their own files.
func (paladin *Paladin) applyHealingTalents() {
	paladin.applyHealingLight()
	paladin.applyInfusionOfLight()
	paladin.applyIllumination()
	paladin.applyHolyPower()
	paladin.applyBoundlessLight()
}

func (paladin *Paladin) applyHealingLight() {
	if paladin.Talents.HealingLight == 0 {
		return
	}
	paladin.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Pct,
		ClassMask:  PaladinSpellMaskHealingLight,
		FloatValue: 1 + healingLightPctPerRank*float64(paladin.Talents.HealingLight),
	})
}

// spiritualFocusPushbackReduction is the chance a spell of this class mask
// keeps its cast time when the paladin is hit; zero for a spell the talent
// does not name.
func (paladin *Paladin) spiritualFocusPushbackReduction(classMask uint64) float64 {
	if classMask&PaladinSpellMaskSpiritualFocus == 0 {
		return 0
	}
	return spiritualFocusPushbackReductionPerRank * float64(paladin.Talents.SpiritualFocus)
}

func (paladin *Paladin) applyInfusionOfLight() {
	rank := paladin.Talents.InfusionOfLight
	if rank == 0 {
		return
	}

	castTime := paladin.AddDynamicMod(core.SpellModConfig{
		Kind:      core.SpellMod_CastTime_Flat,
		ClassMask: PaladinSpellMaskHolyLight,
		TimeValue: -infusionOfLightCastTimePerRank * time.Duration(rank),
	})
	infusion := paladin.RegisterAura(core.Aura{
		Label:    "Infusion of Light",
		ActionID: core.ActionID{SpellID: infusionOfLightActionID},
		Duration: infusionOfLightDuration,
		OnGain: func(*core.Aura, *core.Simulation) {
			castTime.Activate()
		},
		OnExpire: func(*core.Aura, *core.Simulation) {
			castTime.Deactivate()
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.Matches(PaladinSpellMaskHolyLight) {
				aura.Deactivate(sim)
			}
		},
	})

	paladin.RegisterAura(core.Aura{
		Label:    "Infusion of Light Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnHealDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.Matches(PaladinSpellMaskInfusionTriggers) && result.DidCrit() {
				infusion.Activate(sim)
			}
		},
	})
}

// markIlluminating records the base mana cost Illumination returns half of
// when this heal crits. A heal that is a child of a cast (Holy Shock's
// heal, Light's Vigil's heal) has no cost of its own, so the cast's cost
// is passed in.
func (paladin *Paladin) markIlluminating(heal *core.Spell, baseCost float64) {
	if paladin.illuminationBaseCost == nil {
		paladin.illuminationBaseCost = map[*core.Spell]float64{}
	}
	paladin.illuminationBaseCost[heal] = baseCost
}

// applyIllumination: "After getting a critical effect from your Flash of
// Light, Holy Light, Light's Vigil, or Holy Shock heal spell you have a
// 20% chance to gain Mana equal to 50% of the base cost of the spell" at
// rank 1, 20% more a rank.
func (paladin *Paladin) applyIllumination() {
	rank := paladin.Talents.Illumination
	if rank == 0 {
		return
	}

	chance := illuminationChancePerRank * float64(rank)
	manaMetrics := paladin.NewManaMetrics(core.ActionID{SpellID: illuminationActionID})
	// Forever's 20272 also carries effect 1, aura 134 (regeneration while
	// casting) at 50 for 15 s (SpellEffect 1237075, DurationIndex 8,
	// 1.60.1.70707); Wowhead's buff text: "Regenerating at 50% of normal
	// rate while casting spells".
	castingRegen := paladin.RegisterAura(core.Aura{
		Label:    "Illumination Casting Regen",
		ActionID: core.ActionID{SpellID: illuminationRegenSpellID},
		Duration: illuminationRegenDuration,
		OnGain: func(_ *core.Aura, sim *core.Simulation) {
			paladin.PseudoStats.SpiritRegenRateCasting += illuminationRegenWhileCasting
			paladin.UpdateManaRegenRates()
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			paladin.PseudoStats.SpiritRegenRateCasting -= illuminationRegenWhileCasting
			paladin.UpdateManaRegenRates()
		},
	})
	paladin.RegisterAura(core.Aura{
		Label:    "Illumination",
		ActionID: core.ActionID{SpellID: illuminationActionID},
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnHealDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.Matches(PaladinSpellMaskIlluminating) || !result.DidCrit() {
				return
			}
			if !sim.Proc(chance, "Illumination") {
				return
			}
			paladin.AddMana(sim, paladin.illuminationBaseCost[spell]*illuminationManaShareOfBaseCost, manaMetrics)
			castingRegen.Activate(sim)
		},
	})
}

// applyHolyPower raises the crit bonus of the Holy heals.
func (paladin *Paladin) applyHolyPower() {
	rank := float64(paladin.Talents.HolyPower)
	if rank == 0 {
		return
	}
	paladin.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_CritDamageBonus_Flat,
		ClassMask:  PaladinSpellMaskIlluminating,
		FloatValue: holyPowerCritBonusPerRank * rank,
	})
}

// applyBoundlessLight raises Holy Shock's heal crit chance.
func (paladin *Paladin) applyBoundlessLight() {
	rank := float64(paladin.Talents.BoundlessLight)
	if rank == 0 {
		return
	}
	paladin.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Flat,
		ClassMask:  PaladinSpellMaskHolyShockHeal,
		FloatValue: boundlessLightCritPctPerRank * rank * core.CritRatingPerCritChance,
	})
}
