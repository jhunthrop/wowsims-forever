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
// (408439 .. 408477, one per Lightning Bolt rank). Blizzard's 9 November
// hotfix (t/2403040): "Lightning Overload now deals the correct amount of
// damage for Lightning Bolt." Client 1.60.1.70888's SpellEffect rows make
// each one half its bolt rank again (rank 10: 176.5 + 1.05 a level against
// the bolt's 353 + 2.1), after 1.60.1.70755 had raised the bolts alone.
// Client 1.60.1.70938 carries them natively, but the site's spellconst
// rounds EffectBasePointsF to a whole number (36.5 reads 36, 176.5 reads
// 176), so the exact rows stay stated here.
var LightningBoltOverloadSpellId = [LightningBoltRanks + 1]int32{0, 408439, 408440, 408441, 408442, 408443, 408472, 408473, 408474, 408475, 408477}
var LightningBoltOverloadDamage = [LightningBoltRanks + 1]clientdamage.Effect{
	{},
	{Amount: 8, Variance: 0.142857, PerLevel: 0.05, SpellLevel: 1, MaxLevel: 6},
	{Amount: 15, Variance: 0.142857, PerLevel: 0.05, SpellLevel: 8, MaxLevel: 13},
	{Amount: 26, Variance: 0.163265, PerLevel: 0.15, SpellLevel: 14, MaxLevel: 19},
	{Amount: 36.5, Variance: 0.134831, PerLevel: 0.25, SpellLevel: 20, MaxLevel: 25},
	{Amount: 54.5, Variance: 0.134328, PerLevel: 0.35, SpellLevel: 26, MaxLevel: 31},
	{Amount: 74, Variance: 0.120879, PerLevel: 0.5, SpellLevel: 32, MaxLevel: 37},
	{Amount: 97, Variance: 0.125, PerLevel: 0.5, SpellLevel: 38, MaxLevel: 43},
	{Amount: 120, Variance: 0.114094, PerLevel: 0.75, SpellLevel: 44, MaxLevel: 49},
	{Amount: 147.5, Variance: 0.119565, PerLevel: 0.8, SpellLevel: 50, MaxLevel: 55},
	{Amount: 176.5, Variance: 0.113122, PerLevel: 1.05, SpellLevel: 56, MaxLevel: 61},
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
