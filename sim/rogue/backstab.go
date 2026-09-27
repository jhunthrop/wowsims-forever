package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// backstabLearnLevels are Backstab's eight rank learn levels; source:
// 1.60.1.70009 client spell data ("Backstab", ranks 1-8). Each rank's level
// also carries a duplicate id in the 460000s (e.g. 462709 beside rank 1's
// 53); those are a clone of the same rank, not an extra rank -- dropped.
var backstabLearnLevels = []int{4, 12, 20, 28, 36, 44, 52, 60}

// backstabSpellID is Backstab's rank -> spell id, index 0 unused. Rank 8's
// id depends on whether AQ content is included, same as the old code.
var backstabSpellID = [9]int32{0, 53, 2589, 2590, 2591, 8721, 11279, 11280, core.TernaryInt32(core.IncludeAQ, 25300, 11281)}

// backstabFlatDamageBonus is Backstab's rank -> flat damage bonus, index 0
// unused. Only ranks 3, 5, 6 and 8 have a tuned value in this file; ranks
// 1-2 carry rank 3's bonus backward, rank 4 carries rank 3's bonus forward,
// and rank 7 carries rank 6's bonus forward, until real numbers are sourced.
var backstabFlatDamageBonus = [9]float64{0, 32, 32, 32, 32, 60, 90, 90, core.TernaryFloat64(core.IncludeAQ, 150, 140)}

func (rogue *Rogue) registerBackstabSpell() {
	rank := core.HighestRankAtLevel(backstabLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	flatDamageBonus := backstabFlatDamageBonus[rank]
	spellID := backstabSpellID[rank]

	damageMultiplier := 1.5 * []float64{1, 1.04, 1.08, 1.12, 1.16, 1.2}[rogue.Talents.Opportunity]

	rogue.Backstab = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_RogueBackstab,
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       rogue.builderFlags(),

		EnergyCost: core.EnergyCostOptions{
			Cost:   60,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			if !rogue.HasDagger(core.MainHand) {
				return false
			}
			return !rogue.PseudoStats.InFrontOfTarget
		},

		// FOREVER: Improved Backstab is not in the client's trees.
		// BonusCritRating: 10 * core.CritRatingPerCritChance * float64(rogue.Talents.ImprovedBackstab),

		CritDamageBonus: rogue.lethality(),

		DamageMultiplier: damageMultiplier,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			baseDamage := (flatDamageBonus + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target)))
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
