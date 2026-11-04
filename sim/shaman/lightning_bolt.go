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
// to the cap, and the width of the roll (rank 10 rolls 341.8-381.0 at
// level 60, a centre of 353 at level 56 with 2.1 a level, since client
// 1.60.1.70755's push; 196 with 1.2 before). The coefficient below is
// the client's cast time / 3.5 with no downrank penalty (0.714 at rank
// 10; the Classic roll these replaced was 428-477 at 0.857).
// sim/shaman/spellconst_damage_test.go checks every rank against the
// vendored client file.
var LightningBoltDamage = [LightningBoltRanks + 1]clientdamage.Effect{
	{},
	{Amount: 16, Variance: 0.142857, PerLevel: 0.1, SpellLevel: 1, MaxLevel: 6},
	{Amount: 30, Variance: 0.142857, PerLevel: 0.1, SpellLevel: 8, MaxLevel: 13},
	{Amount: 52, Variance: 0.163265, PerLevel: 0.3, SpellLevel: 14, MaxLevel: 19},
	{Amount: 73, Variance: 0.134831, PerLevel: 0.5, SpellLevel: 20, MaxLevel: 25},
	{Amount: 109, Variance: 0.134328, PerLevel: 0.7, SpellLevel: 26, MaxLevel: 31},
	{Amount: 148, Variance: 0.120219, PerLevel: 1, SpellLevel: 32, MaxLevel: 37},
	{Amount: 194, Variance: 0.116183, PerLevel: 1, SpellLevel: 38, MaxLevel: 43},
	{Amount: 240, Variance: 0.113712, PerLevel: 1.5, SpellLevel: 44, MaxLevel: 49},
	{Amount: 295, Variance: 0.11413, PerLevel: 1.6, SpellLevel: 50, MaxLevel: 55},
	{Amount: 353, Variance: 0.108352, PerLevel: 2.1, SpellLevel: 56, MaxLevel: 61},
}
var LightningBoltSpellCoef = [LightningBoltRanks + 1]float64{0, .429, .571, .714, .714, .714, .714, .714, .714, .714, .714}
var LightningBoltCastTime = [LightningBoltRanks + 1]int32{0, 1500, 2000, 2500, 2500, 2500, 2500, 2500, 2500, 2500, 2500}
var LightningBoltManaCost = [LightningBoltRanks + 1]float64{0, 15, 30, 45, 60, 85, 110, 135, 160, 190, 220}
var LightningBoltLevel = [LightningBoltRanks + 1]int{0, 1, 8, 14, 20, 26, 32, 38, 44, 50, 56}

// LightningBoltOverload* are Lightning Overload's own per-rank spells
// (408439 .. 408477, one per Lightning Bolt rank): spellconst/shaman.json's
// roll and coefficient for each. Before client 1.60.1.70755 each was
// exactly half its Lightning Bolt rank's amount and coefficient; the push
// raised every Lightning Bolt rank and left these rows alone (rank 10's
// overload is still 98 + 0.6 a level at 0.357 against the bolt's 353 + 2.1
// at 0.714), so the overload deals its own row, not half the bolt.
var LightningBoltOverloadSpellId = [LightningBoltRanks + 1]int32{0, 408439, 408440, 408441, 408442, 408443, 408472, 408473, 408474, 408475, 408477}
var LightningBoltOverloadDamage = [LightningBoltRanks + 1]clientdamage.Effect{
	{},
	{Amount: 7, Variance: 0.142857, PerLevel: 0.2, SpellLevel: 1, MaxLevel: 6},
	{Amount: 14, Variance: 0.142857, PerLevel: 0.25, SpellLevel: 8, MaxLevel: 13},
	{Amount: 22, Variance: 0.163265, PerLevel: 0.3, SpellLevel: 14, MaxLevel: 19},
	{Amount: 28, Variance: 0.134831, PerLevel: 0.3, SpellLevel: 20, MaxLevel: 25},
	{Amount: 36, Variance: 0.134328, PerLevel: 0.35, SpellLevel: 26, MaxLevel: 31},
	{Amount: 56, Variance: 0.120879, PerLevel: 0.4, SpellLevel: 32, MaxLevel: 37},
	{Amount: 74, Variance: 0.125, PerLevel: 0.4, SpellLevel: 38, MaxLevel: 43},
	{Amount: 81, Variance: 0.114094, PerLevel: 0.5, SpellLevel: 44, MaxLevel: 49},
	{Amount: 89, Variance: 0.119565, PerLevel: 0.5, SpellLevel: 50, MaxLevel: 55},
	{Amount: 98, Variance: 0.113122, PerLevel: 0.6, SpellLevel: 56, MaxLevel: 61},
}
var LightningBoltOverloadSpellCoef = [LightningBoltRanks + 1]float64{0, 0.2145, 0.2855, 0.357, 0.357, 0.357, 0.357, 0.357, 0.357, 0.357, 0.357}

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
	spell.ClassSpellMask = ShamanSpellMaskLightningBolt
	spell.MissileSpeed = 20
	spell.RequiredLevel = level
	spell.Rank = rank
	spell.BonusCoefficient = spellCoeff
	spell.ClientBaseDamage = damage.Range(casterLevel)

	overloadDamage := LightningBoltOverloadDamage[rank]
	overloadCoef := LightningBoltOverloadSpellCoef[rank]

	hit := func(sim *core.Simulation, target *core.Unit, spell *core.Spell, roll clientdamage.Effect) {
		result := spell.CalcDamage(sim, target, roll.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
		spell.WaitTravelTime(sim, func(sim *core.Simulation) {
			spell.DealDamage(sim, result)
		})
	}

	spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		hit(sim, target, spell, damage)

		// Lightning Overload (talents.go): "a second, similar spell ...
		// at no additional cost", the rank's own overload row, through
		// this same spell object so every multiplier the primary hit
		// already has applies identically.
		if shaman.rollLightningOverload(sim) {
			shaman.AtLightningOverloadCoefficient(spell, overloadCoef, func() { hit(sim, target, spell, overloadDamage) })
		}
	}

	return spell
}
