package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const ChainLightningRanks = 4
const ChainLightningTargetCount = int32(3)

var ChainLightningSpellId = [ChainLightningRanks + 1]int32{0, 421, 930, 2860, 10605}

// ChainLightningDamage and ChainLightningSpellCoef are spellconst/
// shaman.json's own roll and "sp_coefficient". Client 1.60.1.70755's
// push retuned every rank: rank 4 (10605) is EffectBasePointsF 510 with
// 3.3 a level (was 123 with 0.8), every rank's coefficient is 0.714
// (0.571, and 0.517 at rank 3, before), the cast is CastingTimeIndex 19,
// 2500 ms (index 5, 2000 ms, before), and the mana cost rose.
// sim/shaman/spellconst_damage_test.go checks the roll, coefficient, cost
// and cast time against the vendored client file.
var ChainLightningDamage = [ChainLightningRanks + 1]clientdamage.Effect{
	{},
	{Amount: 215, Variance: 0.127451, PerLevel: 1.4, SpellLevel: 32, MaxLevel: 37},
	{Amount: 348, Variance: 0.115646, PerLevel: 2.2, SpellLevel: 40, MaxLevel: 45},
	{Amount: 428, Variance: 0.114713, PerLevel: 2.7, SpellLevel: 48, MaxLevel: 53},
	{Amount: 510, Variance: 0.111111, PerLevel: 3.3, SpellLevel: 56, MaxLevel: 61},
}
var ChainLightningSpellCoef = [ChainLightningRanks + 1]float64{0, .714, .714, .714, .714}
var ChainLightningManaCost = [ChainLightningRanks + 1]float64{0, 245, 360, 425, 495} // client 1.60.1.70755 (was 145/200/250/315)
var ChainLightningLevel = [ChainLightningRanks + 1]int{0, 32, 40, 48, 56}

// ChainLightningOverload* are Lightning Overload's own per-rank Chain
// Lightning spells (408479 .. 408484), unchanged by client 1.60.1.70755's
// push (rank 4's is 62 + 1.2 a level at 0.2855 against the chain's 510 +
// 3.3 at 0.714): the overload chain deals its own row.
var ChainLightningOverloadSpellId = [ChainLightningRanks + 1]int32{0, 408479, 408481, 408482, 408484}
var ChainLightningOverloadDamage = [ChainLightningRanks + 1]clientdamage.Effect{
	{},
	{Amount: 44, Variance: 0.137255, PerLevel: 0.75, SpellLevel: 32, MaxLevel: 37},
	{Amount: 50, Variance: 0.122449, PerLevel: 0.9, SpellLevel: 40, MaxLevel: 45},
	{Amount: 56, Variance: 0.115, PerLevel: 1.05, SpellLevel: 48, MaxLevel: 53},
	{Amount: 62, Variance: 0.114943, PerLevel: 1.2, SpellLevel: 56, MaxLevel: 61},
}
var ChainLightningOverloadSpellCoef = [ChainLightningRanks + 1]float64{0, 0.2855, 0.2855, 0.2855, 0.2855}

// ChainLightningCastTime is every rank's CastingTimeIndex 19 (2500 ms)
// and ChainLightningCooldown its category cooldown (6000 ms), client
// 1.60.1.70755.
const (
	ChainLightningCastTime = 2500 * time.Millisecond
	ChainLightningCooldown = 6 * time.Second
)

func (shaman *Shaman) registerChainLightningSpell() {
	shaman.ChainLightning = make([]*core.Spell, ChainLightningRanks+1)

	cdTimer := shaman.NewTimer()

	for rank := 1; rank <= ChainLightningRanks; rank++ {
		config := shaman.newChainLightningSpellConfig(rank, cdTimer)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.ChainLightning[rank] = shaman.RegisterSpell(config)
		}
	}
}

func (shaman *Shaman) newChainLightningSpellConfig(rank int, cdTimer *core.Timer) core.SpellConfig {
	spellId := ChainLightningSpellId[rank]
	damage := ChainLightningDamage[rank]
	casterLevel := int(shaman.Level)
	spellCoeff := ChainLightningSpellCoef[rank]
	overloadDamage := ChainLightningOverloadDamage[rank]
	overloadCoef := ChainLightningOverloadSpellCoef[rank]
	manaCost := ChainLightningManaCost[rank]
	level := ChainLightningLevel[rank]

	cooldown := ChainLightningCooldown
	castTime := ChainLightningCastTime

	shaman.ChainLightningBounceCoefficient = .70 // 30% reduction per bounce
	targetCount := ChainLightningTargetCount

	spell := shaman.newElectricSpellConfig(
		core.ActionID{SpellID: spellId},
		manaCost,
		castTime,
	)

	spell.SpellCode = SpellCode_ShamanChainLightning
	spell.ClassSpellMask = ShamanSpellMaskChainLightning
	spell.RequiredLevel = level
	spell.Rank = rank
	spell.BonusCoefficient = spellCoeff
	spell.ClientBaseDamage = damage.Range(casterLevel)
	spell.Cast.CD = core.Cooldown{
		Timer:    cdTimer,
		Duration: cooldown,
	}

	// Pool-sized ceiling, live-bounded loop; see APLActionMultidot.
	results := make([]*core.SpellResult, min(int(targetCount), len(shaman.Env.Encounter.AllTargetUnits)))

	spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		numHits := min(len(results), len(sim.Encounter.TargetUnits))

		// Shared by the primary bounce chain and Lightning Overload's
		// second one (talents.go's rollLightningOverload, with the rank's
		// own overload roll): the chain starts from the primary target
		// each time, and the damage multiplier it entered with is
		// restored when the chain ends.
		dealBounces := func(roll clientdamage.Effect) {
			enteringMultiplier := spell.DamageMultiplier
			bounceTarget := target
			for hitIndex := 0; hitIndex < numHits; hitIndex++ {
				results[hitIndex] = spell.CalcDamage(sim, bounceTarget, roll.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
				bounceTarget = sim.Environment.NextTargetUnit(bounceTarget)
				spell.DamageMultiplier *= shaman.ChainLightningBounceCoefficient
			}

			for _, result := range results[:numHits] {
				spell.DealDamage(sim, result)
			}

			spell.DamageMultiplier = enteringMultiplier
		}

		dealBounces(damage)

		if shaman.rollLightningOverload(sim) {
			shaman.AtLightningOverloadCoefficient(spell, overloadCoef, func() { dealBounces(overloadDamage) })
		}
	}

	return spell
}
