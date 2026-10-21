package shaman

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const FrostShockRanks = 4

var FrostShockSpellId = [FrostShockRanks + 1]int32{0, 8056, 8058, 10472, 10473}
var FrostShockDamage = [FrostShockRanks + 1]clientdamage.Effect{
	{},
	{Amount: 66, Variance: 0.065217, PerLevel: 0.9, SpellLevel: 20, MaxLevel: 25},
	{Amount: 124, Variance: 0.065728, PerLevel: 1.3, SpellLevel: 34, MaxLevel: 39},
	{Amount: 188, Variance: 0.058309, PerLevel: 1.5, SpellLevel: 46, MaxLevel: 51},
	{Amount: 283, Variance: 0.056, PerLevel: 1.8, SpellLevel: 58, MaxLevel: 63},
}
var FrostShockSpellCoef = [FrostShockRanks + 1]float64{0, .386, .386, .386, .386}
var FrostShockManaCost = [FrostShockRanks + 1]float64{0, 115, 225, 325, 430}
var FrostShockLevel = [FrostShockRanks + 1]int{0, 20, 34, 46, 58}

func (shaman *Shaman) registerFrostShockSpell(shockTimer *core.Timer) {
	shaman.FrostShock = make([]*core.Spell, FrostShockRanks+1)

	for rank := 1; rank <= FrostShockRanks; rank++ {
		config := shaman.newFrostShockSpellConfig(rank, shockTimer)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.FrostShock[rank] = shaman.RegisterSpell(config)
		}
	}
}

func (shaman *Shaman) newFrostShockSpellConfig(rank int, shockTimer *core.Timer) core.SpellConfig {
	spellId := FrostShockSpellId[rank]
	damage := FrostShockDamage[rank]
	casterLevel := int(shaman.Level)
	spellCoeff := FrostShockSpellCoef[rank]
	manaCost := FrostShockManaCost[rank]
	level := FrostShockLevel[rank]

	spell := shaman.newShockSpellConfig(
		core.ActionID{SpellID: spellId},
		core.SpellSchoolFrost,
		manaCost,
		shockTimer,
	)

	spell.SpellCode = SpellCode_ShamanFrostShock
	spell.ClassSpellMask = ShamanSpellMaskFrostShock
	spell.RequiredLevel = level
	spell.Rank = rank
	spell.BonusCoefficient = spellCoeff
	spell.ClientBaseDamage = damage.Range(casterLevel)

	spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
	}

	return spell
}
