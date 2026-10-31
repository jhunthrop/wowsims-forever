package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	// swiftJudgementActionID is the talent's spell (node 110878, spell
	// 1310994). Client 1.60.1.70707: "Unleash the energy of a Seal spell
	// upon an enemy. Does not consume the Seal. Refer to individual Seals
	// for Judgement effect. Does not share a cooldown with your Judgement
	// ability." The 1 minute cooldown, no cost and no global cooldown are
	// the client's row for 1310994 and Wowhead's live Forever tooltip
	// (2026-10-31); the client row still carries the older text.
	swiftJudgementActionID = 1310994
	swiftJudgementCooldown = 60 * time.Second

	// swiftJudgementSpellLevel is the spell's own spell_level (the talent
	// is what gates it).
	swiftJudgementSpellLevel = 1
)

// registerSwiftJudgement is the Swift Judgement talent: a second Judgement
// of the Seal that is up, on its own cooldown. Talents that name
// "Judgement" (Improved Judgement's cooldown, Sanctified Judgement's
// refund) read the Judgement ability, not this one.
func (paladin *Paladin) registerSwiftJudgement() {
	if !paladin.Talents.SwiftJudgement {
		return
	}

	paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: swiftJudgementActionID},
		SpellSchool: core.SpellSchoolHoly,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagNoOnCastComplete | core.SpellFlagCastTimeNoGCD,

		RequiredLevel: swiftJudgementSpellLevel,

		Cast: core.CastConfig{
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: swiftJudgementCooldown,
			},
		},
		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return paladin.currentSeal.IsActive()
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
			paladin.currentJudgement.Cast(sim, target)
		},
	})
}
