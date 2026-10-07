package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// FerociousBiteDamage is constants_auto_gen.go's row as an Effect per rank.
var FerociousBiteDamage = clientdamage.FromTable(FerociousBiteBaseDamage[:], FerociousBitePointsPerLevel[:], FerociousBiteLevel[:], FerociousBiteMaxLevel[:])

// The bite's ladder and its non-combo-point roll are constants_auto_gen.go's
// (FerociousBiteBaseDamage rank 5 is the client's 52-112); the per-combo-point
// and per-energy steps are the Era figures the client's table does not state
// in a form the generator reads.
var ferociousBiteDamagePerComboPoint = [FerociousBiteRanks + 1]float64{0, 36, 59, 92, 128, 147}
var ferociousBiteDamagePerEnergy = [FerociousBiteRanks + 1]float64{0, 1.0, 1.5, 2.0, 2.5, 2.7}

// ferociousBiteLearnLevels is core.HighestRankAtLevel's input shape
// (rank r, 1-based, learned at ferociousBiteLearnLevels[r-1]), read off
// FerociousBiteLevel so the two never drift apart.
var ferociousBiteLearnLevels = FerociousBiteLevel[1:]

func (druid *Druid) registerFerociousBiteSpell() {
	// rank was pinned to the top rank (5, or 4 without AQ - rank V is
	// not available until AQ release) regardless of the character's
	// level, so a levelling druid's Ferocious Bite always registered
	// under the level-60 id (31018) and the rotation's own castSpell
	// for its OWN learned rank (22568/22827/22828 at 32/40/48) could
	// never find it. HighestRankAtLevel picks the rank actually
	// learned, capped at the same AQ-flag ceiling as before.
	maxRank := core.TernaryInt(core.IncludeAQ, 5, 4)
	rank := min(core.HighestRankAtLevel(ferociousBiteLearnLevels, druid.Level), maxRank)
	if rank == 0 {
		return
	}
	config := druid.newFerociousBiteSpellConfig(rank)
	druid.FerociousBite = druid.RegisterSpell(Cat, config)
}

func (druid *Druid) newFerociousBiteSpellConfig(rank int) core.SpellConfig {
	damage := FerociousBiteDamage[rank]
	casterLevel := int(druid.Level)
	damagePerComboPoint := ferociousBiteDamagePerComboPoint[rank]
	damagePerEnergy := ferociousBiteDamagePerEnergy[rank]

	return core.SpellConfig{
		SpellCode:      SpellCode_DruidFerociousBite,
		ClassSpellMask: DruidSpellMaskFerociousBite,
		ActionID:       core.ActionID{SpellID: FerociousBiteSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          SpellFlagOmen | core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		RequiredLevel: FerociousBiteLevel[rank],
		Rank:          rank,

		EnergyCost: core.EnergyCostOptions{
			Cost:   35,
			Refund: 0,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return druid.ComboPoints() > 0
		},

		// FOREVER: Feral Aggression is not in the client's trees.
		// DamageMultiplierAdditive: 1 + 0.03*float64(druid.Talents.FeralAggression),
		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
		BonusCoefficient:         1,
		ClientBaseDamage:         damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.BreakProwl(sim)

			comboPoints := float64(druid.ComboPoints())
			attackPower := spell.MeleeAttackPower(target)
			excessEnergy := druid.CurrentEnergy()

			baseDamage := damage.Roll(sim, casterLevel) +
				damagePerComboPoint*comboPoints +
				attackPower*0.03*comboPoints +
				damagePerEnergy*excessEnergy

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				druid.SpendEnergy(sim, excessEnergy, spell.EnergyMetrics())
				druid.SpendComboPoints(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}
		},
	}
}

func (druid *Druid) CurrentFerociousBiteCost() float64 {
	return druid.FerociousBite.Cost.GetCurrentCost()
}
