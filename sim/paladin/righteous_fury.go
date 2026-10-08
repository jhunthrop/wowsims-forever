package paladin

import (
	"github.com/wowsims/classic/sim/core"
)

const (
	righteousFuryActionID = 25780

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

// registerRighteousFury is Righteous Fury as an option: a paladin with
// PaladinOptions.RighteousFury has it on for the whole fight (it lasts 30
// minutes and costs 30% of base mana to cast, paid before the pull). A
// paladin without it generates less threat if it took Instrument of Law.
func (paladin *Paladin) registerRighteousFury() {
	if !paladin.Options.RighteousFury {
		paladin.applyInstrumentOfLawThreatPenalty()
		return
	}

	paladin.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_Threat_Pct,
		School:     core.SpellSchoolHoly,
		FloatValue: 1 + righteousFuryHolyThreatBonus,
	})

	damageTakenMultiplier := 1 - improvedRighteousFuryDamageTakenPerRank*float64(paladin.Talents.ImprovedRighteousFury)
	paladin.RegisterAura(*core.MakePermanent(&core.Aura{
		Label:    "Righteous Fury",
		ActionID: core.ActionID{SpellID: righteousFuryActionID},
		OnGain: func(aura *core.Aura, _ *core.Simulation) {
			aura.Unit.PseudoStats.DamageTakenMultiplier *= damageTakenMultiplier
		},
		OnExpire: func(aura *core.Aura, _ *core.Simulation) {
			aura.Unit.PseudoStats.DamageTakenMultiplier /= damageTakenMultiplier
		},
	}))
}

func (paladin *Paladin) applyInstrumentOfLawThreatPenalty() {
	rank := paladin.Talents.InstrumentOfLaw
	if rank == 0 {
		return
	}
	paladin.PseudoStats.ThreatMultiplier *= 1 - instrumentOfLawThreatReductionPerRank*float64(rank)
}
