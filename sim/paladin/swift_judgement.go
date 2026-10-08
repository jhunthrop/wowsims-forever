package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	// swiftJudgementActionID is the talent's spell (node 110878, spell
	// 1310994): a 60 second cooldown, no cost, no global cooldown.
	swiftJudgementActionID = 1310994
	swiftJudgementCooldown = 60 * time.Second

	// swiftJudgementSpellLevel is the spell's own spell_level (the talent
	// is what gates it).
	swiftJudgementSpellLevel = 1

	// swiftJudgementCostPct is the spell's cost modifier: "reduces the
	// Mana cost of your next Judgement by 100%".
	swiftJudgementCostPct = -100
)

// registerSwiftJudgement is the Swift Judgement talent (node 110878):
// "Finishes the remaining cooldown on your Judgement ability and reduces
// the Mana cost of your next Judgement by 100%." Research notes it
// "resets Judgement when it misses"; the client text is the cooldown and
// the free cast, which is what this models. The free cast waits for the
// next Judgement however long that takes (the client states no duration
// on the modifier; unconfirmed).
func (paladin *Paladin) registerSwiftJudgement() {
	if !paladin.Talents.SwiftJudgement {
		return
	}

	freeJudgement := paladin.AddDynamicMod(core.SpellModConfig{
		Kind:      core.SpellMod_PowerCost_Pct,
		ClassMask: PaladinSpellMaskJudgement,
		IntValue:  swiftJudgementCostPct,
	})

	paladin.swiftJudgementAura = paladin.RegisterAura(core.Aura{
		Label:    "Swift Judgement",
		ActionID: core.ActionID{SpellID: swiftJudgementActionID},
		Duration: core.NeverExpires,
		OnGain: func(*core.Aura, *core.Simulation) {
			freeJudgement.Activate()
		},
		OnExpire: func(*core.Aura, *core.Simulation) {
			freeJudgement.Deactivate()
		},
	})

	paladin.RegisterSpell(core.SpellConfig{
		ActionID: core.ActionID{SpellID: swiftJudgementActionID},
		Flags:    core.SpellFlagAPL | core.SpellFlagNoOnCastComplete,

		RequiredLevel: swiftJudgementSpellLevel,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: swiftJudgementCooldown,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			paladin.judgement.CD.Reset()
			paladin.swiftJudgementAura.Activate(sim)
		},
	})
}

// spendSwiftJudgement ends the free Judgement once a Judgement has been
// cast; it is a no-op without the talent.
func (paladin *Paladin) spendSwiftJudgement(sim *core.Simulation) {
	if paladin.swiftJudgementAura != nil {
		paladin.swiftJudgementAura.Deactivate(sim)
	}
}
