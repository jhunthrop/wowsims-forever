package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// garroteLearnLevels are Garrote's six rank learn levels; source:
// 1.60.1.70009 client spell data ("Garrote", ranks 1-6). Each rank's level
// also carries a duplicate id in the 460000s (e.g. 462724 beside rank 1's
// 703); those are a clone of the same rank, not an extra rank -- dropped.
var garroteLearnLevels = []int{14, 22, 30, 38, 46, 54}

// garroteSpellID is Garrote's rank -> spell id, index 0 unused.
var garroteSpellID = [7]int32{0, 703, 8631, 8632, 8633, 11289, 11290}

func (rogue *Rogue) registerGarrote() {
	rank := core.HighestRankAtLevel(garroteLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	tickDamage := GarroteTickDamage[rank]
	casterLevel := int(rogue.Level)
	spellID := garroteSpellID[rank]

	rogue.Garrote = rogue.GetOrRegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueGarrote,
		ActionID:      core.ActionID{SpellID: spellID},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         SpellFlagBuilder | core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		RequiredLevel: garroteLearnLevels[rank-1],

		EnergyCost: core.EnergyCostOptions{
			Cost:   50.0 - 10*float64(rogue.Talents.DirtyDeeds),
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			if !rogue.IsStealthed() {
				return false
			}
			return !rogue.PseudoStats.InFrontOfTarget
		},

		DamageMultiplier: opportunityMultiplier[rankIndex(rogue.Talents.Opportunity, opportunityMultiplier[:])],
		ThreatMultiplier: 1,
		ClientBaseDamage: tickDamage.Range(casterLevel),

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Garrote",
			},
			NumberOfTicks: 6,
			TickLength:    time.Second * 3,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				damage := tickDamage.Roll(sim, casterLevel) + dot.Spell.MeleeAttackPower(target)*0.03
				dot.Snapshot(target, damage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialNoBlockDodgeParryNoCritNoHitCounter)
			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
				spell.Dot(target).Apply(sim)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},
	})
}
