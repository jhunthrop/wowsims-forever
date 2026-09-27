package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// shredLearnLevels are Shred's five rank learn levels; source: 1.60.1.70009
// client spell data ("Shred", ranks 1-5; the level-20 rank-0 id is an
// internal copy, not a player rank).
var shredLearnLevels = []int{22, 30, 38, 46, 54}

// shredSpellID is Shred's rank -> spell id, index 0 unused.
var shredSpellID = [6]int32{0, 5221, 6800, 8992, 9829, 9830}

// shredFlatDamageBonus is Shred's rank -> flat damage bonus, index 0 unused.
// Rank 2 (id 6800) has no tuned value in this file; it carries rank 1's
// bonus forward until a real number is sourced.
var shredFlatDamageBonus = [6]float64{0, 24, 24, 44, 64, 80}

func (druid *Druid) registerShredSpell() {
	rank := core.HighestRankAtLevel(shredLearnLevels, druid.Level)
	if rank == 0 {
		return
	}

	damageMultiplier := 2.25
	flatDamageBonus := shredFlatDamageBonus[rank]

	druid.Shred = druid.RegisterSpell(Cat, core.SpellConfig{
		SpellCode:   SpellCode_DruidShred,
		ActionID:    core.ActionID{SpellID: shredSpellID[rank]},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOmen | SpellFlagBuilder,

		EnergyCost: core.EnergyCostOptions{
			// FOREVER: Improved Shred is not in the client's trees.
			// Cost:   60 - 6*float64(druid.Talents.ImprovedShred),
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
			return !druid.PseudoStats.InFrontOfTarget
		},

		DamageMultiplier: damageMultiplier,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := flatDamageBonus + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				druid.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
		ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			baseDamage := flatDamageBonus + spell.Unit.AutoAttacks.MH().CalculateAverageWeaponDamage(spell.MeleeAttackPower(target))

			baseres := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeExpectedMagicAlwaysHit)

			attackTable := spell.Unit.AttackTables[target.UnitIndex][spell.CastType]
			critChance := spell.PhysicalCritChance(attackTable)
			critMod := critChance * (spell.CritMultiplier(attackTable) - 1)

			baseres.Damage *= 1 + critMod

			return baseres
		},
	})
}
