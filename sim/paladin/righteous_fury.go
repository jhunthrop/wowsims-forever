package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	righteousFuryActionID = 25780

	// righteousFuryLevel, righteousFuryBaseManaCost (30% of base mana) and
	// righteousFuryDuration (1800000 ms) are spell 25780's required level,
	// cost_pct and duration in the client.
	righteousFuryLevel        = 16
	righteousFuryBaseManaCost = 0.30
	righteousFuryDuration     = 30 * time.Minute

	// righteousFuryHolyThreatBonus is Righteous Fury's effect 0 (spell
	// 25780, aura 10 "modify threat", amount 60, school mask 2): +60%
	// threat on Holy school damage. The client states no other school, so
	// white hits (physical) are not raised.
	righteousFuryHolyThreatBonus = 0.60

	// improvedRighteousFuryDamageTakenPerRank is Improved Righteous Fury
	// (node 105634): "While Righteous Fury is active, all damage taken is
	// reduced by 2%", 4% and 6% at ranks 2 and 3. It is a damage-taken
	// reduction, not the vanilla 16/33/50% threat multiplier: Forever's
	// Righteous Fury has no threat talent.
	improvedRighteousFuryDamageTakenPerRank = 0.02

	// instrumentOfLawThreatReductionPerRank is Instrument of Law (node
	// 110880): "reduces all threat you generate by 10% while Righteous Fury
	// is not active", 20% at rank 2 (its cast-time half is in
	// applyDeclarativeTalents).
	instrumentOfLawThreatReductionPerRank = 0.10
)

// registerRighteousFury registers Righteous Fury as an aura, as a castable
// spell, and as an option: a paladin with PaladinOptions.RighteousFury has
// it on for the whole fight without casting (it lasts 30 minutes and costs
// 30% of base mana to cast, paid before the pull).
func (paladin *Paladin) registerRighteousFury() {
	instrumentOfLaw := newInstrumentOfLawPenalty(paladin)
	if !paladin.Options.RighteousFury {
		instrumentOfLaw.set(true)
	}

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
			instrumentOfLaw.set(false)
			aura.Unit.PseudoStats.DamageTakenMultiplier *= damageTakenMultiplier
		},
		OnExpire: func(aura *core.Aura, _ *core.Simulation) {
			holyThreat.Deactivate()
			instrumentOfLaw.set(true)
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

// instrumentOfLawPenalty is Instrument of Law's threat reduction, which
// holds only while Righteous Fury is not active.
type instrumentOfLawPenalty struct {
	unit       *core.Unit
	multiplier float64
	applied    bool
}

func newInstrumentOfLawPenalty(paladin *Paladin) *instrumentOfLawPenalty {
	rank := paladin.Talents.InstrumentOfLaw
	return &instrumentOfLawPenalty{
		unit:       &paladin.Unit,
		multiplier: 1 - instrumentOfLawThreatReductionPerRank*float64(rank),
	}
}

// set applies or lifts the penalty; setting the state it is in is a no-op.
func (penalty *instrumentOfLawPenalty) set(active bool) {
	if penalty.applied == active {
		return
	}
	penalty.applied = active
	if active {
		penalty.unit.PseudoStats.ThreatMultiplier *= penalty.multiplier
	} else {
		penalty.unit.PseudoStats.ThreatMultiplier /= penalty.multiplier
	}
}
