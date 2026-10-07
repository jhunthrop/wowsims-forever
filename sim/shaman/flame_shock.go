package shaman

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const FlameShockRanks = 6

var FlameShockSpellId = [FlameShockRanks + 1]int32{0, 8050, 8052, 8053, 10447, 10448, 29228}

// The direct hit's damage and coefficient, and the per-tick damage and
// coefficient of the 12 s DoT's four 3 s ticks, are spellconst/shaman.json's
// own: rank 6 is 166 at 0.214 up front and 44 at 0.1 a tick (176 over the
// DoT), where the Classic numbers these replaced were 292 and 320 over the
// DoT, with rank 1-2 coefficients below the client's 0.214.
var FlameShockBaseDamage = [FlameShockRanks + 1]float64{0, 20, 33, 43, 82, 128, 166}
var FlameShockTickDamage = [FlameShockRanks + 1]float64{0, 7, 8, 14, 21, 34, 44}
var FlameShockBaseSpellCoef = [FlameShockRanks + 1]float64{0, .214, .214, .214, .214, .214, .214}
var FlameShockDotSpellCoef = [FlameShockRanks + 1]float64{0, .1, .1, .1, .1, .1, .1}
var FlameShockManaCost = [FlameShockRanks + 1]float64{0, 55, 95, 160, 250, 345, 410}
var FlameShockLevel = [FlameShockRanks + 1]int{0, 10, 18, 28, 40, 52, 60}

func (shaman *Shaman) registerFlameShockSpell(shockTimer *core.Timer) {
	shaman.FlameShock = make([]*core.Spell, FlameShockRanks+1)

	for rank := 1; rank <= FlameShockRanks; rank++ {
		if FlameShockLevel[rank] <= int(shaman.Level) {
			shaman.FlameShock[rank] = shaman.RegisterSpell(shaman.newFlameShockSpell(rank, shockTimer))
		}
	}
}

func (shaman *Shaman) newFlameShockSpell(rank int, shockTimer *core.Timer) core.SpellConfig {
	numTicks := 4
	tickDuration := time.Second * 3

	spellId := FlameShockSpellId[rank]
	baseDamage := FlameShockBaseDamage[rank]
	baseDotDamage := FlameShockTickDamage[rank]
	baseSpellCoeff := FlameShockBaseSpellCoef[rank]
	dotSpellCoeff := FlameShockDotSpellCoef[rank]
	manaCost := FlameShockManaCost[rank]
	level := FlameShockLevel[rank]

	spell := shaman.newShockSpellConfig(
		core.ActionID{SpellID: spellId},
		core.SpellSchoolFire,
		manaCost,
		shockTimer,
	)

	spell.SpellCode = SpellCode_ShamanFlameShock
	spell.RequiredLevel = level
	spell.Rank = rank

	spell.Cast.IgnoreHaste = true

	spell.BonusCoefficient = baseSpellCoeff

	// Call of Flame's tooltip (talents/shaman.json node 104770) names
	// "Fire Totems and ... Flame Shock, Fire Nova, and Lava Burst"
	// explicitly; Lava Burst and the fire totems already apply this
	// multiplicatively via shaman.callOfFlameMultiplier() (lava_burst.go,
	// fire_totems.go), but Flame Shock was left on newShockSpellConfig's
	// flat DamageMultiplier: 1. AttackerDamageMultiplier multiplies
	// spell.DamageMultiplier into both the initial hit
	// (calcDamageInternal) and the DoT's own snapshot
	// (dot.Snapshot -> AttackerDamageMultiplier), so setting it here
	// covers both without touching the DotConfig.
	spell.DamageMultiplier = shaman.callOfFlameMultiplier()

	spell.Dot = core.DotConfig{
		Aura: core.Aura{
			Label: fmt.Sprintf("Flame Shock (Rank %d)", rank),
		},

		NumberOfTicks:    int32(numTicks),
		TickLength:       tickDuration,
		BonusCoefficient: dotSpellCoeff,

		OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
			dot.Snapshot(target, baseDotDamage, isRollover)
		},

		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
		},
	}

	spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
		if result.Landed() {
			spell.Dot(result.Target).Apply(sim)
		}
	}

	return spell
}
