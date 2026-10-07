package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (paladin *Paladin) registerDivineFavor() {
	if !paladin.Talents.DivineFavor {
		return
	}

	var affectedSpells []*core.Spell
	paladin.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Matches(PaladinSpellMaskDivineFavor) {
			affectedSpells = append(affectedSpells, spell)
		}
	})

	cd := core.Cooldown{
		Timer:    paladin.NewTimer(),
		Duration: time.Minute * 2,
	}

	// spend removes the buff and puts the ability on cooldown.
	spend := func(aura *core.Aura, sim *core.Simulation) {
		aura.Deactivate(sim)
		cd.Set(sim.CurrentTime + cd.Duration)
		paladin.UpdateMajorCooldowns()
	}

	aura := paladin.RegisterAura(core.Aura{
		Label:    "Divine Favor",
		ActionID: core.ActionID{SpellID: 20216},
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			core.Each(affectedSpells, func(spell *core.Spell) {
				spell.BonusCritRating += core.CritRatingPerCritChance * 100
			})
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			core.Each(affectedSpells, func(spell *core.Spell) {
				spell.BonusCritRating -= core.CritRatingPerCritChance * 100
			})
		},
		// Whichever of the named spells lands first spends the buff: a
		// Holy Shock that hits an enemy, or a heal.
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
			if spell.SpellCode == SpellCode_PaladinHolyShock {
				spend(aura, sim)
			}
		},
		OnHealDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
			if spell.Matches(PaladinSpellMaskHealingLight) {
				spend(aura, sim)
			}
		},
	})

	divineFavor := paladin.RegisterSpell(core.SpellConfig{
		ActionID: aura.ActionID,
		Flags:    core.SpellFlagNoOnCastComplete,
		// 4% of base mana: the client prices this through
		// SpellPower.PowerCostPct (spellconst cost_pct), not the flat
		// cost column, which reads 0.
		ManaCost: core.ManaCostOptions{
			BaseCost: DivineFavorManaCostPct[0] / 100,
		},
		Cast: core.CastConfig{
			CD: cd,
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			aura.Activate(sim)
		},
	})
	paladin.AddMajorCooldown(core.MajorCooldown{
		Spell: divineFavor,
		Type:  core.CooldownTypeDPS,
	})
}
