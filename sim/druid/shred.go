package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// shredLearnLevels are Shred's five rank learn levels: client 1.60.1.70707
// SpellLevels and SkillLineAbility (rank 1, 5221, at 20; Era's 22 until
// Blizzard's 15 October beta change, "Rank 1 Shred is learned at level 20
// instead of 22").
var shredLearnLevels = []int{20, 30, 38, 46, 54}

// shredSpellID is Shred's rank -> spell id, index 0 unused.
var shredSpellID = [6]int32{0, 5221, 6800, 8992, 9829, 9830}

// shredFlatDamageBonus is Shred's rank -> flat damage bonus (effect 0, the
// weapon damage bonus), index 0 unused. Source: 1.60.1.70707 client
// SpellEffect.
var shredFlatDamageBonus = [6]float64{0, 22, 32, 44, 64, 80}

// shredWeaponDamageMultiplier is Shred's weapon damage percentage on every
// rank: client effect 1 is 200 since the 1.60.1.70755 server push 113403
// (155 before it, which beta logs matched: a rank 1 Shred over a 31-damage
// paw hit averaged 82 = 1.55 x (31 + 22); Era's was 225).
const shredWeaponDamageMultiplier = 2.00

func (druid *Druid) registerShredSpell() {
	rank := core.HighestRankAtLevel(shredLearnLevels, druid.Level)
	if rank == 0 {
		return
	}

	flatDamageBonus := shredFlatDamageBonus[rank]

	druid.Shred = druid.RegisterSpell(Cat, core.SpellConfig{
		SpellCode:      SpellCode_DruidShred,
		ClassSpellMask: DruidSpellMaskShred,
		ActionID:       core.ActionID{SpellID: shredSpellID[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOmen | SpellFlagBuilder,

		RequiredLevel: shredLearnLevels[rank-1],

		EnergyCost: core.EnergyCostOptions{
			// Shredding Attacks (node 104945, proto field
			// shredding_attacks): "Reduces the Energy cost of your
			// Shred ability by 6/12/18." The Lacerate half of the same
			// talent is not modeled: Bear Form's own damage kit is not
			// modeled in this package (see RegisterFeralCatSpells's
			// comment above).
			Cost:   60 - 6*float64(clampRank(druid.Talents.ShreddingAttacks, 3)),
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

		// Savage Fury's client mask holds Shred (0x8000).
		DamageMultiplierAdditive: druid.savageFuryDamageMultiplier(),
		DamageMultiplier:         shredWeaponDamageMultiplier,
		ThreatMultiplier:         1,
		BonusCoefficient:         1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.BreakProwl(sim)

			baseDamage := flatDamageBonus + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			baseDamage *= druid.RendAndTearMultiplier(target)

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				druid.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
		ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			baseDamage := flatDamageBonus + spell.Unit.AutoAttacks.MH().CalculateAverageWeaponDamage(spell.MeleeAttackPower(target))
			baseDamage *= druid.RendAndTearMultiplier(target)

			baseres := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeExpectedMagicAlwaysHit)

			attackTable := spell.Unit.AttackTables[target.UnitIndex][spell.CastType]
			critChance := spell.PhysicalCritChance(attackTable)
			critMod := critChance * (spell.CritMultiplier(attackTable) - 1)

			baseres.Damage *= 1 + critMod

			return baseres
		},
	})
}
