package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// cheapShotSpellID, cheapShotLevel and cheapShotComboPoints are the client's
// Cheap Shot (1.60.1.70009 spell 1833): level 26, 60 energy, a stun and
// effect 30 (energize, misc 4: combo points) of 2. The stun is not modelled,
// no encounter target can be stunned; the combo points are the result.
const (
	cheapShotSpellID     int32 = 1833
	cheapShotLevel       int   = 26
	cheapShotEnergyCost        = 60.0
	cheapShotComboPoints int32 = 2
)

func (rogue *Rogue) registerCheapShot() {
	if rogue.Level < int32(cheapShotLevel) {
		return
	}

	rogue.CheapShot = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueCheapShot,
		ActionID:      core.ActionID{SpellID: cheapShotSpellID},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         rogue.builderFlags(),
		RequiredLevel: cheapShotLevel,

		EnergyCost: core.EnergyCostOptions{
			Cost:   cheapShotEnergyCost,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.IsStealthed() && !rogue.PseudoStats.InFrontOfTarget
		},

		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialNoBlockDodgeParryNoCritNoHitCounter)
			if result.Landed() {
				rogue.AddComboPoints(sim, cheapShotComboPoints, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},
	})
}
