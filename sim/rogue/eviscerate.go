package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// eviscerateLearnLevels are Eviscerate's nine rank learn levels; source:
// 1.60.1.70009 client spell data ("Eviscerate", ranks 1-9; the level-1
// rank-0 ids are internal copies, not player ranks).
var eviscerateLearnLevels = []int{1, 8, 16, 24, 32, 40, 48, 56, 60}

// eviscerateSpellID is Eviscerate's rank -> spell id, index 0 unused.
// Rank 9 only exists with AQ content, same as the old code's ternary.
var eviscerateSpellID = [10]int32{0, 2098, 6760, 6761, 6762, 8623, 8624, 11299, 11300, core.TernaryInt32(core.IncludeAQ, 31016, 11300)}

// eviscerateFlatDamage/ComboDamageBonus/DamageVariance are Eviscerate's
// rank -> damage terms, index 0 unused. Only ranks 4, 6, 7 and 9 have a
// tuned value in this file; ranks 1-3 carry rank 4's terms backward, rank 5
// carries rank 4's terms forward, and rank 8 carries rank 7's terms
// forward, until real numbers are sourced.
var eviscerateFlatDamage = [10]float64{0, 10, 10, 10, 10, 10, 22, 34, 34, core.TernaryFloat64(core.IncludeAQ, 54, 48)}
var eviscerateComboDamageBonus = [10]float64{0, 31, 31, 31, 31, 31, 77, 110, 110, core.TernaryFloat64(core.IncludeAQ, 170, 151)}
var eviscerateDamageVariance = [10]float64{0, 20, 20, 20, 20, 20, 44, 68, 68, core.TernaryFloat64(core.IncludeAQ, 108, 96)}

func (rogue *Rogue) registerEviscerate() {
	rank := core.HighestRankAtLevel(eviscerateLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	flatDamage := eviscerateFlatDamage[rank]
	comboDamageBonus := eviscerateComboDamageBonus[rank]
	damageVariance := eviscerateDamageVariance[rank]
	spellID := eviscerateSpellID[rank]

	rogue.Eviscerate = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueEviscerate,
		ActionID:      core.ActionID{SpellID: spellID},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         rogue.finisherFlags() | SpellFlagColdBlooded,
		MetricSplits:  6,
		RequiredLevel: eviscerateLearnLevels[rank-1],

		EnergyCost: core.EnergyCostOptions{
			Cost:   35,
			Refund: 0,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(spell.Unit.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		DamageMultiplier: 1 +
			[]float64{0, 0.05, 0.10, 0.15}[rogue.Talents.ImprovedEviscerate] +
			[]float64{0, 0.02, 0.04, 0.06}[rogue.Talents.Aggression],
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			comboPoints := rogue.ComboPoints()
			flatBaseDamage := flatDamage + comboDamageBonus*float64(comboPoints)

			baseDamage := sim.Roll(flatBaseDamage, flatBaseDamage+damageVariance) +
				0.03*float64(comboPoints)*spell.MeleeAttackPower(target)

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				rogue.SpendComboPoints(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}

			spell.DealDamage(sim, result)
		},
	})
	rogue.Finishers = append(rogue.Finishers, rogue.Eviscerate)
}
