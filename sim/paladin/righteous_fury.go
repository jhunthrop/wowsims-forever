package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	righteousFuryActionID = 25780

	// righteousFuryLevel, righteousFuryBaseManaCost (30% of base mana) and
	// righteousFuryDuration (1800000 ms) are spell 25780's required level,
	// cost_pct and duration in the client. Client 1.60.1.70707 moved it
	// to level 10 (was 16).
	righteousFuryLevel        = 10
	righteousFuryBaseManaCost = 0.30
	righteousFuryDuration     = 30 * time.Minute

	// righteousFuryHolyThreatBonus is Righteous Fury's effect 0 (spell
	// 25780, aura 10 "modify threat", amount 60, school mask 2): +60%
	// threat on Holy school damage. The client states no other school, so
	// white hits (physical) are not raised.
	righteousFuryHolyThreatBonus = 0.60

	// improvedRighteousFuryDamageTakenPerRank is Improved Righteous Fury
	// (node 105634): "While Righteous Fury is active, all damage taken is
	// reduced by 3%", 6% and 9% at ranks 2 and 3 (client 1.60.1.70707; 2%
	// a rank before). It is a damage-taken
	// reduction, not the vanilla 16/33/50% threat multiplier: Forever's
	// Righteous Fury has no threat talent.
	improvedRighteousFuryDamageTakenPerRank = 0.03
)

// registerRighteousFury registers Righteous Fury as an aura, as a castable
// spell, and as an option: a paladin with PaladinOptions.RighteousFury has
// it on for the whole fight without casting (it lasts 30 minutes and costs
// 30% of base mana to cast, paid before the pull).
func (paladin *Paladin) registerRighteousFury() {
	holyThreat := paladin.AddDynamicMod(core.SpellModConfig{
		Kind:       core.SpellMod_Threat_Pct,
		School:     core.SpellSchoolHoly,
		FloatValue: 1 + righteousFuryHolyThreatBonus,
	})
	damageTakenMultiplier := 1 - improvedRighteousFuryDamageTakenPerRank*float64(paladin.Talents.ImprovedRighteousFury)
	aura := paladin.RegisterAura(core.Aura{
		Label:    "Righteous Fury",
		ActionID: core.ActionID{SpellID: righteousFuryActionID},
		Duration: righteousFuryDuration,
		OnGain: func(aura *core.Aura, _ *core.Simulation) {
			holyThreat.Activate()
			aura.Unit.PseudoStats.DamageTakenMultiplier *= damageTakenMultiplier
		},
		OnExpire: func(aura *core.Aura, _ *core.Simulation) {
			holyThreat.Deactivate()
			aura.Unit.PseudoStats.DamageTakenMultiplier /= damageTakenMultiplier
		},
	})
	if paladin.Options.RighteousFury {
		core.MakePermanent(aura)
	}

	if paladin.Level >= righteousFuryLevel {
		paladin.registerRighteousFuryCast(aura)
	}
}

func (paladin *Paladin) registerRighteousFuryCast(aura *core.Aura) {
	spell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: righteousFuryActionID},
		SpellSchool: core.SpellSchoolHoly,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagAPL | core.SpellFlagNoOnCastComplete,

		RequiredLevel: righteousFuryLevel,

		ManaCost: core.ManaCostOptions{BaseCost: righteousFuryBaseManaCost},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			aura.Activate(sim)
		},
	})
	spell.RelatedSelfBuff = aura
}
