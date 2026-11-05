package druid

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Bloom (Restoration node 104918, tier 2, spell 1328621, client
// 1.60.1.70707 hotfix): "When your Rejuvenation or Regrowth spell expires due
// to its duration, the target is instantly healed for an amount equal to 100%
// of your Spirit. If either spell expires due to being dispelled, the target
// is healed for 50% of your Spirit instead." The heal is the hidden spell
// 1328622 ("Bloom"). Nothing in these fights dispels a druid's heal, so only
// the full-duration half is reached; a Rejuvenation or Regrowth that
// Swiftmend consumes, or that a recast refreshes, did not expire by its
// duration and blooms nothing.
//
// The heal is triggered, not cast, so it rolls no cast or spell procs: no
// OnCastComplete (Vestments of the Devout's "your spellcasts" mana) and an
// empty proc mask (Omen of Clarity's 4% a spell was fitted per logged cast),
// as the engine's other triggered spells do.
//
// The heal can critically strike: its SpellMisc row (1328622) lacks
// SPELL_ATTR2_CANT_CRIT, which the client's triggered heals that cannot
// crit carry (Judgement of Light 20267, Living Seed 414683).
//
// The amount is Spirit only (client 1.60.1.70790). The heal effect (SpellEffect
// for 1328622, index 0, effect 10) has EffectBonusCoefficient 0, so healing
// power adds nothing; BloomSpellCoeff's 0.2143 in constants_auto_gen.go is
// the generator's convention fill for a zero column and is deliberately not
// read. Its SpellClassMask is 0 in every word, so family-masked talents
// (Gift of Nature, Improved Rejuvenation) do not reach it: the spell has no
// ClassSpellMask. Its SpellMisc Attributes_3 (0x40010000) lacks
// SPELL_ATTR3_IGNORE_CASTER_MODIFIERS (0x20000000), so unmasked caster and
// target healing multipliers still apply.
const (
	bloomSpellID        = 1328622
	bloomSpiritFraction = 1.0
)

// registerBloomSpell registers Bloom's heal when the druid has the talent.
func (druid *Druid) registerBloomSpell() {
	if !druid.Talents.Bloom {
		return
	}
	druid.Bloom = druid.RegisterSpell(Any, core.SpellConfig{
		ActionID:    core.ActionID{SpellID: bloomSpellID},
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagHelpful | core.SpellFlagPassiveSpell | core.SpellFlagNoOnCastComplete,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealHealing(sim, target, bloomHealing(druid.GetStat(stats.Spirit)), spell.OutcomeHealingCrit)
		},
	})
}

// bloomHealing is the heal of a Rejuvenation or Regrowth that ran its full
// duration, for a druid with the given Spirit.
func bloomHealing(spirit float64) float64 {
	return bloomSpiritFraction * spirit
}

// bloomOnExpire is the heal over time aura's OnExpire for Rejuvenation and
// Regrowth: when the aura ran out on its own (not consumed or cancelled
// early), Bloom heals its target. Aura.Deactivate clears the expiry before
// OnExpire runs, so the full run is read off the start and the duration.
func (druid *Druid) bloomOnExpire(aura *core.Aura, sim *core.Simulation) {
	if druid.Bloom == nil || sim.CurrentTime < aura.StartedAt()+aura.Duration {
		return
	}
	druid.Bloom.Cast(sim, aura.Unit)
}
