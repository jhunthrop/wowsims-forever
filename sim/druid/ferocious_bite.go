package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// FerociousBiteDamage is constants_auto_gen.go's row as an Effect per rank.
var FerociousBiteDamage = clientdamage.FromTable(FerociousBiteBaseDamage[:], FerociousBitePointsPerLevel[:], FerociousBiteLevel[:], FerociousBiteMaxLevel[:])

// The bite's ladder and its non-combo-point roll are constants_auto_gen.go's
// (FerociousBiteBaseDamage rank 6 is the client's 102 +/- 35.7%). The per-energy
// step is the client's dummy effect 1 (amount 70/120/180/250/310/330 hundredths
// per energy, pinned by spellconst_damage_test.go); the per-combo-point step is
// effect 0's EffectPointsPerResource, which spellconst does not carry
// (30/45/74/115/160/181; all three from the 1.60.1.70755 server push 113403). Rank 1
// (1328618, level 24) is the server's hotfix push, which the client tables
// carry since the site's pipeline layers the push (round 6); its 24 a combo
// point is Wowhead's live tooltip (33-43 at 1 point over the client's 9-19,
// read 2026-11-01).
//
// Predatory Discipline (Feral node 104951, spell 1328613, client 1.60.1.70707
// hotfix): "Your Ferocious Bite ability no longer converts extra Energy into
// damage, but still deals extra damage based on your excess Energy." The
// bite keeps the per-energy damage on the energy left after its cost but no
// longer spends that energy.
var ferociousBiteDamagePerComboPoint = [FerociousBiteRanks + 1]float64{0, 30, 45, 74, 115, 160, 181}
var ferociousBiteDamagePerEnergy = [FerociousBiteRanks + 1]float64{0, 0.7, 1.2, 1.8, 2.5, 3.1, 3.3}

// ferociousBiteLearnLevels is core.HighestRankAtLevel's input shape
// (rank r, 1-based, learned at ferociousBiteLearnLevels[r-1]), read off
// FerociousBiteLevel so the two never drift apart.
var ferociousBiteLearnLevels = FerociousBiteLevel[1:]

func (druid *Druid) registerFerociousBiteSpell() {
	// rank was pinned to the top rank (6, or 5 without AQ - the top rank
	// is not available until AQ release) regardless of the character's
	// level, so a levelling druid's Ferocious Bite always registered
	// under the level-60 id (31018) and the rotation's own castSpell
	// for its OWN learned rank (22568/22827/22828 at 32/40/48) could
	// never find it. HighestRankAtLevel picks the rank actually
	// learned, capped at the same AQ-flag ceiling as before.
	maxRank := core.TernaryInt(core.IncludeAQ, FerociousBiteRanks, FerociousBiteRanks-1)
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
				attackPower*ferociousBiteAttackPowerShare(comboPoints) +
				damagePerEnergy*excessEnergy
			baseDamage *= druid.RendAndTearMultiplier(target)

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				if !druid.Talents.PredatoryDiscipline {
					druid.SpendEnergy(sim, excessEnergy, spell.EnergyMetrics())
				}
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

// ferociousBiteAttackPowerPerPoint is Ferocious Bite's share of attack power
// a combo point: Era's 3%, raised "about 25%" with the base points by
// Blizzard's Forever notes of 3 November 2026 (the client's per-point step
// went 145 to 181 at rank 6 in the same push).
const ferociousBiteAttackPowerPerPoint = 0.0375

// ferociousBiteAttackPowerShare is the attack power share of a bite at the
// given combo points.
func ferociousBiteAttackPowerShare(comboPoints float64) float64 {
	return ferociousBiteAttackPowerPerPoint * comboPoints
}
