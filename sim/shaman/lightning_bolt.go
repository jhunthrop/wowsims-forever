package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const LightningBoltRanks = 10

var LightningBoltSpellId = [LightningBoltRanks + 1]int32{0, 403, 529, 548, 915, 943, 6041, 10391, 10392, 15207, 15208}

// LightningBoltDamage is spellconst/shaman.json's own roll for ids 403
// through 15208: the centre at the spell's level, the per-level growth
// to the cap, and the width of the roll (rank 10 rolls 189.9-211.7 at
// level 60, a centre of 196 at level 56). The coefficient below is the
// client's cast time / 3.5 with no downrank penalty (0.714 at rank 10;
// the Classic roll these replaced was 428-477 at 0.857).
// sim/shaman/spellconst_damage_test.go checks every rank against the
// vendored client file.
var LightningBoltDamage = [LightningBoltRanks + 1]clientdamage.Effect{
	{},
	{Amount: 14, Variance: 0.142857, PerLevel: 0.4, SpellLevel: 1, MaxLevel: 6},
	{Amount: 29, Variance: 0.142857, PerLevel: 0.5, SpellLevel: 8, MaxLevel: 13},
	{Amount: 45, Variance: 0.163265, PerLevel: 0.6, SpellLevel: 14, MaxLevel: 19},
	{Amount: 56, Variance: 0.134831, PerLevel: 0.6, SpellLevel: 20, MaxLevel: 25},
	{Amount: 72, Variance: 0.134328, PerLevel: 0.7, SpellLevel: 26, MaxLevel: 31},
	{Amount: 111, Variance: 0.120219, PerLevel: 0.8, SpellLevel: 32, MaxLevel: 37},
	{Amount: 147, Variance: 0.116183, PerLevel: 0.8, SpellLevel: 38, MaxLevel: 43},
	{Amount: 162, Variance: 0.113712, PerLevel: 1, SpellLevel: 44, MaxLevel: 49},
	{Amount: 178, Variance: 0.11413, PerLevel: 1, SpellLevel: 50, MaxLevel: 55},
	{Amount: 196, Variance: 0.108352, PerLevel: 1.2, SpellLevel: 56, MaxLevel: 61},
}
var LightningBoltSpellCoef = [LightningBoltRanks + 1]float64{0, .429, .571, .714, .714, .714, .714, .714, .714, .714, .714}
var LightningBoltCastTime = [LightningBoltRanks + 1]int32{0, 1500, 2000, 2500, 2500, 2500, 2500, 2500, 2500, 2500, 2500}
var LightningBoltManaCost = [LightningBoltRanks + 1]float64{0, 15, 30, 45, 60, 85, 110, 135, 160, 190, 220}
var LightningBoltLevel = [LightningBoltRanks + 1]int{0, 1, 8, 14, 20, 26, 32, 38, 44, 50, 56}

func (shaman *Shaman) registerLightningBoltSpell() {
	shaman.LightningBolt = make([]*core.Spell, LightningBoltRanks+1)

	for rank := 1; rank <= LightningBoltRanks; rank++ {
		config := shaman.newLightningBoltSpellConfig(rank)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.LightningBolt[rank] = shaman.RegisterSpell(config)
		}
	}
}

func (shaman *Shaman) newLightningBoltSpellConfig(rank int) core.SpellConfig {
	spellId := LightningBoltSpellId[rank]
	damage := LightningBoltDamage[rank]
	casterLevel := int(shaman.Level)
	spellCoeff := LightningBoltSpellCoef[rank]
	castTime := LightningBoltCastTime[rank]
	manaCost := LightningBoltManaCost[rank]
	level := LightningBoltLevel[rank]

	spell := shaman.newElectricSpellConfig(
		core.ActionID{SpellID: spellId},
		manaCost,
		time.Millisecond*time.Duration(castTime),
	)
	spell.SpellCode = SpellCode_ShamanLightningBolt
	spell.MissileSpeed = 20
	spell.RequiredLevel = level
	spell.Rank = rank
	spell.BonusCoefficient = spellCoeff
	spell.ClientBaseDamage = damage.Range(casterLevel)

	hit := func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
		spell.WaitTravelTime(sim, func(sim *core.Simulation) {
			spell.DealDamage(sim, result)
		})
	}

	spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		hit(sim, target, spell)

		// Lightning Overload (talents.go): "a second, similar spell ...
		// at no additional cost that causes half damage", through this
		// same spell object so every multiplier the primary hit already
		// has applies identically.
		if shaman.rollLightningOverload(sim) {
			shaman.AtLightningOverloadScale(spell, func() { hit(sim, target, spell) })
		}
	}

	return spell
}
