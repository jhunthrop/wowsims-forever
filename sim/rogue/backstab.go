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

func (rogue *Rogue) registerBackstabSpell() {
	rank := core.HighestRankAtLevel(backstabLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	flatDamage := BackstabDamage[rank]
	casterLevel := int(rogue.Level)
	spellID := backstabSpellID[rank]

	damageMultiplier := 1.5 * opportunityMultiplier[rankIndex(rogue.Talents.Opportunity, opportunityMultiplier[:])]

	rogue.Backstab = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueBackstab,
		ActionID:      core.ActionID{SpellID: spellID},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         rogue.builderFlags() | SpellFlagColdBlooded,
		RequiredLevel: backstabLearnLevels[rank-1],

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
		ClientBaseDamage: flatDamage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			baseDamage := flatDamage.Roll(sim, casterLevel) + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
