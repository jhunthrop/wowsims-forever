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

	// Spiritual Focus (node 20205): "a 35% chance to not lose casting time
	// when you take damage" a rank. The engine's name for that chance is
	// the spell's pushback reduction.
	spiritualFocusPushbackReductionPerRank = 0.35

	// Infusion of Light (node 426065): a Holy Shock or Flash of Light crit
	// "reduce[s] the cast time of your next Holy Light cast within 15 sec
	// by 0.5 sec" a rank.
	infusionOfLightActionID        = 426065
	infusionOfLightDuration        = 15 * time.Second
	infusionOfLightCastTimePerRank = 500 * time.Millisecond

	// Illumination (node 20210): a 20% chance a rank, on a heal crit, to
	// gain 50% of the spell's base mana cost.
	illuminationActionID            = 20210
	illuminationChancePerRank       = 0.20
	illuminationManaShareOfBaseCost = 0.50

	// Holy Power (node 5923): Holy Shock and Holy Strike crit chance 3% a
	// rank, every other spell 1%.
	holyPowerSpecialCritPctPerRank = 3.0
	holyPowerOtherCritPctPerRank   = 1.0
)

// applyHealingTalents is the healing-relevant part of the Holy tree. Divine
// Favor and Light's Vigil are spells and live in their own files.
func (paladin *Paladin) applyHealingTalents() {
	paladin.applyHealingLight()
	paladin.applyInfusionOfLight()
	paladin.applyIllumination()
	paladin.applyHolyPower()
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
		},
	})
}

// applyHolyPower: "Increases the critical strike chance of your Holy Shock
// and Holy Strike spells by 3%, and all other spells by 1%" a rank. Every
// Holy spell (the heals included) gets the 1%, and the named two get the
// remaining 2% on top.
func (paladin *Paladin) applyHolyPower() {
	rank := float64(paladin.Talents.HolyPower)
	if rank == 0 {
		return
	}
	paladin.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Flat,
		School:     core.SpellSchoolHoly,
		FloatValue: holyPowerOtherCritPctPerRank * rank * core.CritRatingPerCritChance,
	})
	paladin.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Flat,
		ClassMask:  PaladinSpellMaskHolyPowerSpecials,
		FloatValue: (holyPowerSpecialCritPctPerRank - holyPowerOtherCritPctPerRank) * rank * core.CritRatingPerCritChance,
	})
}
