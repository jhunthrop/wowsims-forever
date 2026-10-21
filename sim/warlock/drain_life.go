package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const DrainLifeRanks = 6

// DrainLifeTickDamage is spellconst/warlock.json's own per-tick amount for
// the drain's periodic health-leech effect on ids 689 through 11700
// (10/14/22/28/39/51 at period 1 s over 5 ticks, no growth). Ranks 2-6 used
// to be 17/29/41/55/71, 10-45% high: closest to a stale pre-Forever classic
// tooltip total over 5.
var DrainLifeTickDamage = [DrainLifeRanks + 1]clientdamage.Effect{
	{},
	{Amount: 10, SpellLevel: 14, MaxLevel: 19},
	{Amount: 14, SpellLevel: 22, MaxLevel: 27},
	{Amount: 22, SpellLevel: 30, MaxLevel: 35},
	{Amount: 28, SpellLevel: 38, MaxLevel: 43},
	{Amount: 39, SpellLevel: 46, MaxLevel: 51},
	{Amount: 51, SpellLevel: 54, MaxLevel: 59},
}

func (warlock *Warlock) getDrainLifeBaseConfig(rank int) core.SpellConfig {
	numTicks := int32(5)

	spellId := [DrainLifeRanks + 1]int32{0, 689, 699, 709, 7651, 11699, 11700}[rank]
	spellCoeff := [DrainLifeRanks + 1]float64{0, .1, .1, .1, .1, .1, .1}[rank]
	damage := DrainLifeTickDamage[rank]
	casterLevel := int(warlock.Level)
	baseDamage := damage.Center(casterLevel)
	manaCost := [DrainLifeRanks + 1]float64{0, 55, 85, 135, 185, 240, 300}[rank]
	level := [DrainLifeRanks + 1]int{0, 14, 22, 30, 38, 46, 54}[rank]

	actionID := core.ActionID{SpellID: spellId}

	healingSpell := warlock.GetOrRegisterSpell(core.SpellConfig{
		ActionID:    actionID.WithTag(1),
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagPassiveSpell | core.SpellFlagHelpful,

		DamageMultiplier: 1,
		ThreatMultiplier: 0,
	})

	spellConfig := core.SpellConfig{
		ActionID:    actionID,
		SpellSchool: core.SpellSchoolShadow,
		SpellCode:   SpellCode_WarlockDrainLife,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagAPL | core.SpellFlagResetAttackSwing | WarlockFlagAffliction | core.SpellFlagChanneled,

		RequiredLevel:    level,
		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "DrainLife-" + warlock.Label + strconv.Itoa(rank),
			},
			NumberOfTicks:    numTicks,
			TickLength:       1 * time.Second,
			BonusCoefficient: spellCoeff,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
				// Drain Life heals so it snapshots target modifiers
				// Update 2024-06-29: It no longer snapshots on PTR
				// dot.SnapshotAttackerMultiplier *= dot.Spell.TargetDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex][dot.Spell.CastType], true)
				if !isRollover {
					// Soul Siphon (talents.go): "+4/8/12% per each of
					// your other Affliction effects active on the
					// target". Evaluated once per cast, same as every
					// other snapshot field here, rather than per tick.
					dot.SnapshotAttackerMultiplier *= warlock.soulSiphonMultiplier(target, dot.Spell)
				}
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				result := dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				health := result.Damage
				healingSpell.CalcAndDealHealing(sim, healingSpell.Unit, health, healingSpell.OutcomeHealing)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {

				dot := spell.Dot(target)
				dot.Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},
		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			if useSnapshot {
				dot := spell.Dot(target)
				return dot.CalcSnapshotDamage(sim, target, spell.OutcomeExpectedMagicAlwaysHit)
			} else {
				return spell.CalcPeriodicDamage(sim, target, baseDamage, spell.OutcomeExpectedMagicAlwaysHit)
			}
		},
	}

	return spellConfig
}

func (warlock *Warlock) registerDrainLifeSpell() {
	warlock.DrainLife = make([]*core.Spell, 0)
	for rank := 1; rank <= DrainLifeRanks; rank++ {
		config := warlock.getDrainLifeBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.DrainLife = append(warlock.DrainLife, warlock.GetOrRegisterSpell(config))
		}
	}
}
