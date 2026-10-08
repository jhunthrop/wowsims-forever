package shaman

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const EarthShockRanks = 7

var EarthShockSpellId = [EarthShockRanks + 1]int32{0, 8042, 8044, 8045, 8046, 10412, 10413, 10414}

// The ids above are the player ranks (8042...10414), whose spellconst
// entries state the roll below and a 0.386 coefficient; the 408681-408690
// and 1220744-1220751 entries are the taunting tank variants (effect 114,
// "taunts the target") whose larger numbers the Classic roll these
// replaced sat close to.
var EarthShockDamage = [EarthShockRanks + 1]clientdamage.Effect{
	{},
	{Amount: 18, Variance: 0.111111, PerLevel: 0.5, SpellLevel: 4, MaxLevel: 9},
	{Amount: 33, Variance: 0.060606, PerLevel: 0.7, SpellLevel: 8, MaxLevel: 13},
	{Amount: 49, Variance: 0.064516, PerLevel: 0.9, SpellLevel: 14, MaxLevel: 19},
	{Amount: 81, Variance: 0.065041, PerLevel: 1.1, SpellLevel: 24, MaxLevel: 29},
	{Amount: 132, Variance: 0.060345, PerLevel: 1.3, SpellLevel: 36, MaxLevel: 41},
	{Amount: 205, Variance: 0.059459, PerLevel: 1.6, SpellLevel: 48, MaxLevel: 53},
	{Amount: 301, Variance: 0.052731, PerLevel: 1.9, SpellLevel: 60, MaxLevel: 65},
}
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
	damage := EarthShockDamage[rank]
	casterLevel := int(shaman.Level)
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
	spell.ClassSpellMask = ShamanSpellMaskEarthShock
	spell.RequiredLevel = level
	spell.Rank = rank

	spell.ThreatMultiplier = 2
	spell.BonusCoefficient = spellCoeff
	spell.ClientBaseDamage = damage.Range(casterLevel)

	spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
	}

	return spell
}
