package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// RakeInitialDamage is constants_auto_gen.go's row as an Effect per rank.
var RakeInitialDamage = clientdamage.FromTable(RakeBaseDamage[:], RakePointsPerLevel[:], RakeLevel[:], RakeMaxLevel[:])

// The Rake ladder is constants_auto_gen.go's (RakeSpellId, RakeLevel and
// RakeBaseDamage hold the client's amounts: rank 4 is 61 up front, where the
// Era ladder this replaced had 58). The client states no width and no
// per-level growth, and its tick is not a generated column, so
// RakeTickDamage is spellconst/druid.json's periodic effect for the same
// ids (34 a tick at rank 4, where the Era ladder had 32).
var RakeTickDamage = [RakeRanks + 1]clientdamage.Effect{
	{},
	{Amount: 16, SpellLevel: 24},
	{Amount: 21, SpellLevel: 34},
	{Amount: 26, SpellLevel: 44},
	{Amount: 34, SpellLevel: 54},
}

func (druid *Druid) registerRakeSpell() {
	// Add highest available rake rank for level.
	for rank := RakeRanks; rank >= 1; rank-- {
		if druid.Level >= int32(RakeLevel[rank]) {
			config := druid.newRakeSpellConfig(rank)
			druid.Rake = druid.RegisterSpell(Cat, config)
			return
		}
	}
}

func (druid *Druid) newRakeSpellConfig(rank int) core.SpellConfig {
	initialDamage := RakeInitialDamage[rank]
	tickDamage := RakeTickDamage[rank]
	casterLevel := int(druid.Level)
	energyCost := 40 - float64(druid.Talents.Ferocity)

	return core.SpellConfig{
		SpellCode:      SpellCode_DruidRake,
		ClassSpellMask: DruidSpellMaskRake,
		ActionID:       core.ActionID{SpellID: RakeSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagIgnoreResists | core.SpellFlagBinary | core.SpellFlagAPL | SpellFlagOmen | SpellFlagBuilder,

		RequiredLevel: RakeLevel[rank],
		Rank:          rank,

		EnergyCost: core.EnergyCostOptions{
			Cost:   energyCost,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},

		ClientBaseDamage: initialDamage.Range(casterLevel),

		DamageMultiplierAdditive: 1 + 0.1*float64(druid.Talents.SavageFury),
		DamageMultiplier:         1,
		ThreatMultiplier:         1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rake",
			},
			NumberOfTicks: 3,
			TickLength:    time.Second * 3,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, tickDamage.Center(casterLevel), isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.BreakProwl(sim)

			result := spell.CalcAndDealDamage(sim, target, initialDamage.Roll(sim, casterLevel), spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				druid.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
				spell.Dot(target).Apply(sim)
			} else {
				spell.IssueRefund(sim)
			}
		},

		ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			initial := spell.CalcPeriodicDamage(sim, target, initialDamage.Center(casterLevel), spell.OutcomeExpectedMagicAlwaysHit)

			attackTable := spell.Unit.AttackTables[target.UnitIndex][spell.CastType]
			critChance := spell.PhysicalCritChance(attackTable)
			critMod := critChance * (spell.CritMultiplier(attackTable) - 1)
			initial.Damage *= 1 + critMod
			return initial
		},
		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			ticks := spell.CalcPeriodicDamage(sim, target, tickDamage.Center(casterLevel), spell.OutcomeExpectedMagicAlwaysHit)
			return ticks
		},
	}
}

func (druid *Druid) CurrentRakeCost() float64 {
	return druid.Rake.Cost.GetCurrentCost()
}
