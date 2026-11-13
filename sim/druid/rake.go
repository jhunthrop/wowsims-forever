package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// RakeInitialDamage is constants_auto_gen.go's row as an Effect per rank.
var RakeInitialDamage = clientdamage.FromTable(RakeBaseDamage[:], RakePointsPerLevel[:], RakeLevel[:], RakeMaxLevel[:])

// The Rake ladder is constants_auto_gen.go's (RakeSpellId, RakeLevel and
// RakeBaseDamage hold the client's amounts: rank 4 is 76 up front after the 70755
// push 113403, 61 before it). The client states no width and no
// per-level growth, and its tick is not a generated column, so
// RakeTickDamage is spellconst/druid.json's periodic effect for the same
// ids (42 a tick at rank 4 after push 113403, 34 before it).
//
// Both halves scale on the server. Blizzard's Forever notes of 3 November
// 2026: "Rake: Base points and attack power scaling increased by about
// 25%" (the base half is the push above). The first Forever cat logs
// (foreverlogs.gg, 5 to 11 November 2026; sim-accuracy 2026-11-12 feral,
// C1) tick the bleed for the rank's amount plus 0.070 to 0.089 of the cat's
// attack power (rakeTickAttackPowerShare, four Feral cats; a Balance druid
// in Cat Form read 0.052). The hit takes the same attack power share on the
// rank's own amount and no weapon damage (sim-accuracy 2026-11-13
// feral-60): every non-crit hit of one cat in our advanced logs is the same
// number (61 on 20 hits at level 29 while its white swings rolled 56 to 69
// before armor; Valdo's on foreverlogs 48 to 49), the hit sits the ranks'
// 7 (times the cat's multipliers) above the tick on every logged cat, and
// both move together when attack power moves (Daigoro: +2 each). The live
// tooltip (9904) renders neither share, as Rip's and Ferocious Bite's
// attack power shares are not rendered either.
//
// unconfirmed: the share's exact value, measured on five logged cats whose
// attack power is read off their stats; a level 60 cat log with known gear
// would pin it.
var RakeTickDamage = [RakeRanks + 1]clientdamage.Effect{
	{},
	{Amount: 20, SpellLevel: 24},
	{Amount: 26, SpellLevel: 34},
	{Amount: 32, SpellLevel: 44},
	{Amount: 42, SpellLevel: 54},
}

// rakeTickAttackPowerShare is the attack power each Rake tick adds to the
// rank's amount (0.24 over the three ticks), the logs' fit above.
const rakeTickAttackPowerShare = 0.08

// rakeHitDamage is Rake's hit before multipliers: the rank's roll plus the
// attack power share a tick takes.
func rakeHitDamage(roll, attackPower float64) float64 {
	return roll + rakeTickAttackPowerShare*attackPower
}

// rakeTickDamage is one Rake tick before multipliers: the rank's amount
// plus the attack power share.
func rakeTickDamage(center, attackPower float64) float64 {
	return center + rakeTickAttackPowerShare*attackPower
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

		DamageMultiplierAdditive: druid.savageFuryDamageMultiplier(),
		DamageMultiplier:         1,
		ThreatMultiplier:         1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rake",
			},
			NumberOfTicks: 3,
			TickLength:    time.Second * 3,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				// Rend and Tear on the bleed when the target already bled as it
				// was applied (bleedRendAndTear, feral_talents.go).
				tick := rakeTickDamage(tickDamage.Center(casterLevel), dot.Spell.MeleeAttackPower(target))
				dot.Snapshot(target, tick*druid.bleedRendAndTear, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.BreakProwl(sim)

			rendAndTear := druid.RendAndTearMultiplier(target)
			baseDamage := rakeHitDamage(initialDamage.Roll(sim, casterLevel), spell.MeleeAttackPower(target)) * rendAndTear
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				druid.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
				druid.bleedRendAndTear = rendAndTear
				spell.Dot(target).Apply(sim)
			} else {
				spell.IssueRefund(sim)
			}
		},

		ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			hit := rakeHitDamage(initialDamage.Center(casterLevel), spell.MeleeAttackPower(target))
			initial := spell.CalcPeriodicDamage(sim, target, hit, spell.OutcomeExpectedMagicAlwaysHit)

			attackTable := spell.Unit.AttackTables[target.UnitIndex][spell.CastType]
			critChance := spell.PhysicalCritChance(attackTable)
			critMod := critChance * (spell.CritMultiplier(attackTable) - 1)
			initial.Damage *= 1 + critMod
			return initial
		},
		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			ticks := spell.CalcPeriodicDamage(sim, target, rakeTickDamage(tickDamage.Center(casterLevel), spell.MeleeAttackPower(target)), spell.OutcomeExpectedMagicAlwaysHit)
			return ticks
		},
	}
}

func (druid *Druid) CurrentRakeCost() float64 {
	return druid.Rake.Cost.GetCurrentCost()
}
