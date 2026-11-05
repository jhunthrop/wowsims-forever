package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// feintLevel is Feint's trainer level (trainables 1966, level 16).
const feintLevel = 16

func (rogue *Rogue) registerFeintSpell() {
	// Nothing checks RequiredLevel at cast time, so a rogue below the trainer
	// level must not have the spell at all.
	if rogue.Level < feintLevel {
		return
	}
	rogue.Feint = rogue.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 1966},
		ClassSpellMask: RogueSpellMaskFeint,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMH,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		// Always registers rank 1 (1966) regardless of character level -
		// every rank shares the same cost/cooldown, only the required
		// level differs, and this engine does not scale Feint by rank.
		RequiredLevel: feintLevel,

		EnergyCost: core.EnergyCostOptions{
			Cost: 20,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Second * 10,
			},
			IgnoreHaste: true,
		},

		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			spell.CalcAndDealOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
		},
	})
}
