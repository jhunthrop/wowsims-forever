package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const ImmolateRanks = 8
const ImmolateCastTime = time.Millisecond * 2000

// ImmolateDamage and ImmolateTickDamage are spellconst/warlock.json's own
// direct hit and per-tick amounts for ids 348 through 25309 (rank 8: a
// direct centre of 158 at level 60, growing 1.2 a level to level 65 before
// that, and 55 a tick, period 3 s, with no growth) - see shadowbolt.go's
// comment on the classic numbers they replaced. The direct hit carries no
// width.
var ImmolateDamage = [ImmolateRanks + 1]clientdamage.Effect{
	{},
	{Amount: 8, PerLevel: 0.7, SpellLevel: 1, MaxLevel: 5},
	{Amount: 17, PerLevel: 0.8, SpellLevel: 10, MaxLevel: 15},
	{Amount: 32, PerLevel: 1.2, SpellLevel: 20, MaxLevel: 25},
	{Amount: 56, PerLevel: 1.5, SpellLevel: 30, MaxLevel: 35},
	{Amount: 72, PerLevel: 1.6, SpellLevel: 40, MaxLevel: 45},
	{Amount: 106, PerLevel: 1.9, SpellLevel: 50, MaxLevel: 55},
	{Amount: 146, PerLevel: 2.3, SpellLevel: 60, MaxLevel: 65},
	{Amount: 158, PerLevel: 2.3, SpellLevel: 60, MaxLevel: 65},
}

var ImmolateTickDamage = [ImmolateRanks + 1]clientdamage.Effect{
	{},
	{Amount: 3, SpellLevel: 1, MaxLevel: 5},
	{Amount: 6, SpellLevel: 10, MaxLevel: 15},
	{Amount: 12, SpellLevel: 20, MaxLevel: 25},
	{Amount: 19, SpellLevel: 30, MaxLevel: 35},
	{Amount: 25, SpellLevel: 40, MaxLevel: 45},
	{Amount: 38, SpellLevel: 50, MaxLevel: 55},
	{Amount: 52, SpellLevel: 60, MaxLevel: 65},
	{Amount: 55, SpellLevel: 60, MaxLevel: 65},
}

func (warlock *Warlock) getImmolateConfig(rank int) core.SpellConfig {
	directCoeff := [ImmolateRanks + 1]float64{0, .2, .2, .2, .2, .2, .2, .2, .2}[rank]
	dotCoeff := [ImmolateRanks + 1]float64{0, .13, .13, .13, .13, .13, .13, .13, .13}[rank]
	damage := ImmolateDamage[rank]
	casterLevel := int(warlock.Level)
	dotDamage := ImmolateTickDamage[rank].Center(casterLevel)
	spellId := [ImmolateRanks + 1]int32{0, 348, 707, 1094, 2941, 11665, 11667, 11668, 25309}[rank]
	manaCost := [ImmolateRanks + 1]float64{0, 25, 45, 90, 155, 220, 295, 370, 380}[rank]
	level := [ImmolateRanks + 1]int{0, 1, 10, 20, 30, 40, 50, 60, 60}[rank]

	return core.SpellConfig{
		SpellCode:   SpellCode_WarlockImmolate,
		ActionID:    core.ActionID{SpellID: spellId},
		SpellSchool: core.SpellSchoolFire,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagBinary | WarlockFlagDestruction,

		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),
		RequiredLevel:    level,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: ImmolateCastTime,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				cast.CastTime = spell.CastTime()
			},
			CastTime: func(spell *core.Spell) time.Duration {
				durationDecrease := time.Duration(0)
				return spell.DefaultCast.CastTime - durationDecrease
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: directCoeff,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Immolate-" + warlock.Label + strconv.Itoa(rank),
			},

			NumberOfTicks:    5,
			TickLength:       time.Second * 3,
			BonusCoefficient: dotCoeff,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, dotDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				var result *core.SpellResult
				result = dot.CalcSnapshotDamage(sim, target, dot.OutcomeTick)
				dot.Spell.DealPeriodicDamage(sim, result)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			oldMultiplier := spell.DamageMultiplier
			// Aftermath (talents.go): "Increases the initial damage of
			// your Immolate spell by 10/20/30/40/50%." Immolate's own
			// initial hit, not its periodic tick - the same multiplier
			// slot improvedImmolateBonus (a talent the client's trees
			// don't have) already uses for the same reason.
			spell.DamageMultiplier *= 1 + warlock.improvedImmolateBonus() + warlock.aftermathInitialDamageBonus()
			result := spell.CalcDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
			spell.DamageMultiplier = oldMultiplier

			if result.Landed() {
				dot := spell.Dot(target)
				dot.Apply(sim)
			}

			spell.DealDamage(sim, result)
		},
		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			if useSnapshot {
				dot := spell.Dot(target)
				return dot.CalcSnapshotDamage(sim, target, dot.Spell.OutcomeExpectedMagicAlwaysHit)
			} else {
				return spell.CalcPeriodicDamage(sim, target, dotDamage, spell.OutcomeExpectedMagicAlwaysHit)
			}
		},
	}
}

func (warlock *Warlock) getActiveImmolateSpell(target *core.Unit) *core.Spell {
	for _, immolateSpell := range warlock.Immolate {
		if immolateSpell.Dot(target).IsActive() {
			return immolateSpell
		}
	}
	return nil
}

func (warlock *Warlock) registerImmolateSpell() {
	warlock.Immolate = make([]*core.Spell, 0)

	maxRank := core.TernaryInt(core.IncludeAQ, ImmolateRanks, ImmolateRanks-1)
	for rank := 1; rank <= maxRank; rank++ {
		config := warlock.getImmolateConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.Immolate = append(warlock.Immolate, warlock.GetOrRegisterSpell(config))
		}
	}
}
