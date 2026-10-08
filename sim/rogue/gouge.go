package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// gougeLearnLevels are Gouge's five rank learn levels; source: 1.60.1.70009
// client spell data ("Gouge", ranks 1-5, GougeLevel in constants_auto_gen.go
// without its unused slot 0).
var gougeLearnLevels = GougeLevel[1:]

// gougeComboPoints is effect 30 (energize, misc 4: combo points) of every
// Gouge rank. The incapacitate is not modelled, no encounter target can be
// stunned; the damage and the combo point are the result.
const gougeComboPoints int32 = 1

func (rogue *Rogue) registerGouge() {
	rank := core.HighestRankAtLevel(gougeLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	damage := GougeDamage[rank]
	casterLevel := int(rogue.Level)

	rogue.Gouge = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueGouge,
		ActionID:      core.ActionID{SpellID: GougeSpellId[rank]},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         rogue.builderFlags(),
		RequiredLevel: GougeLevel[rank],

		EnergyCost: core.EnergyCostOptions{
			Cost:   GougeManaCost[rank], // the client's cost column; the energy this spell spends
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Duration(GougeCooldownMS[rank]) * time.Millisecond,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: GougeSpellCoeff[rank],
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			result := spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMeleeSpecialHitAndCrit)
			if result.Landed() {
				rogue.AddComboPoints(sim, gougeComboPoints, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
