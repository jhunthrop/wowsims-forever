package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (paladin *Paladin) registerJudgement() {
	// Judgement functions as a dummy spell in vanilla.
	// It rolls on the spell hit table and can only miss or hit.
	// Individual seals have their own effects that this spell triggers,
	// that are handled in the implementations of the seal auras.
	paladin.judgement = paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 20271},
		SpellSchool: core.SpellSchoolHoly,
		ProcMask:    core.ProcMaskEmpty,
		// core.SpellFlagPassiveSpell used to sit here too. That flag
		// tells core/metrics_aggregator.go's addSpellMetrics to drop
		// the spell's OWN Casts count ("applied as a result of another
		// spell"), which is correct for the seal-specific judgement
		// spells this ApplyEffects triggers (castSpecificJudgement,
		// below) but wrong for THIS wrapper: it is the ability the
		// rotation names directly (SpellFlagAPL, right beside it), so
		// its Casts count is a real player cast, not a side effect.
		// The identical, identically-caused bug is sim/warrior/execute.go's.
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagNoOnCastComplete | core.SpellFlagCastTimeNoGCD,

		// RequiredLevel 4 and zero mana cost: source 1.60.1.70009 client
		// spell data (spellconst/paladin.json, spell 20271 "Judgement"),
		// cost 0/cost_type 0, spell_level 4. This spell used to carry a
		// BaseCost of 0.06 (6% of base mana), which matched neither the
		// client's own stated cost nor real 1.12 Classic Judgement, which
		// is free -- gated only by its cooldown, never by mana. Flagged
		// by sim/core/testdata/conformance/paladin.golden.md's own
		// "cost 0.00->X.XX" row for every tested level before this fix.
		RequiredLevel: 4,

		Cast: core.CastConfig{
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: time.Second * (10 - time.Duration(paladin.Talents.ImprovedJudgement)),
			},
		},
		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return paladin.currentSeal.IsActive()
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
			paladin.castSpecificJudgement(sim, target, paladin.currentJudgement, paladin.currentSeal)
		},
	})
}

// Helper Function For casting Judgement
func (paladin *Paladin) castSpecificJudgement(sim *core.Simulation, target *core.Unit, judgementSpell *core.Spell, matchingSeal *core.Aura) {
	judgementSpell.Cast(sim, target)
	matchingSeal.Deactivate(sim)
}
