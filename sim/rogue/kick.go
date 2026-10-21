package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// kickLearnLevels are Kick's four rank learn levels; source: 1.60.1.70009
// client spell data ("Kick", ranks 1-4, KickLevel in constants_auto_gen.go
// without its unused slot 0).
var kickLearnLevels = KickLevel[1:]

func (rogue *Rogue) registerKick() {
	rank := core.HighestRankAtLevel(kickLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	damage := KickDamage[rank]
	casterLevel := int(rogue.Level)

	// The interrupt and the lockout it starts are not modelled: encounter
	// targets cast nothing. The damage and the energy are the result.
	rogue.Kick = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueKick,
		ActionID:      core.ActionID{SpellID: KickSpellId[rank]},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		RequiredLevel: KickLevel[rank],

		EnergyCost: core.EnergyCostOptions{
			Cost:   KickManaCost[rank], // the client's cost column; the energy this spell spends
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Duration(KickCooldownMS[rank]) * time.Millisecond,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			result := spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMeleeSpecialHitAndCrit)
			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
