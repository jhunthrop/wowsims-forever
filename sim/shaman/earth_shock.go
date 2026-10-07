package shaman

import (
	"github.com/wowsims/classic/sim/core"
)

const EarthShockRanks = 7

var EarthShockSpellId = [EarthShockRanks + 1]int32{0, 8042, 8044, 8045, 8046, 10412, 10413, 10414}

// The ids above are the player ranks (8042...10414), whose spellconst
// entries read one flat amount and a 0.386 coefficient; the 408681-408690
// and 1220744-1220751 entries are the taunting tank variants (effect 114,
// "taunts the target") whose larger numbers the Classic roll these
// replaced sat close to.
var EarthShockBaseDamage = [EarthShockRanks + 1]float64{0, 18, 33, 49, 81, 132, 205, 301}
var EarthShockSpellCoef = [EarthShockRanks + 1]float64{0, .386, .386, .386, .386, .386, .386, .386}
var EarthShockManaCost = [EarthShockRanks + 1]float64{0, 30, 50, 85, 145, 240, 345, 450}
var EarthShockLevel = [EarthShockRanks + 1]int{0, 4, 8, 14, 24, 36, 48, 60}

func (shaman *Shaman) registerEarthShockSpell(shockTimer *core.Timer) {
	shaman.EarthShock = make([]*core.Spell, EarthShockRanks+1)

	for rank := 1; rank <= EarthShockRanks; rank++ {
		config := shaman.newEarthShockSpellConfig(rank, shockTimer)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.EarthShock[rank] = shaman.RegisterSpell(config)
		}
	}
}

func (shaman *Shaman) newEarthShockSpellConfig(rank int, shockTimer *core.Timer) core.SpellConfig {
	spellId := EarthShockSpellId[rank]
	baseDamage := EarthShockBaseDamage[rank]
	spellCoeff := EarthShockSpellCoef[rank]
	manaCost := EarthShockManaCost[rank]
	level := EarthShockLevel[rank]

	spell := shaman.newShockSpellConfig(
		core.ActionID{SpellID: spellId},
		core.SpellSchoolNature,
		manaCost,
		shockTimer,
	)

	spell.Flags |= core.SpellFlagBinary

	spell.SpellCode = SpellCode_ShamanEarthShock
	spell.RequiredLevel = level
	spell.Rank = rank

	spell.ThreatMultiplier = 2
	spell.BonusCoefficient = spellCoeff

	spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
	}

	return spell
}
