package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const ChainLightningRanks = 4
const ChainLightningTargetCount = int32(3)

var ChainLightningSpellId = [ChainLightningRanks + 1]int32{0, 421, 930, 2860, 10605}

// ChainLightningBaseDamage and ChainLightningSpellCoef are spellconst/
// shaman.json's flat per-rank "amount" and "sp_coefficient" (rank 4 is
// 123 at 0.571, against the Classic roll of 505-564 at 0.714 these
// replaced). The client's rank 3 coefficient reads 0.517, which looks
// like a transposed 0.571, but the client wins and it is registered as
// stated.
var ChainLightningBaseDamage = [ChainLightningRanks + 1]float64{0, 88, 100, 112, 123}
var ChainLightningSpellCoef = [ChainLightningRanks + 1]float64{0, .571, .571, .517, .571}
var ChainLightningManaCost = [ChainLightningRanks + 1]float64{0, 225, 305, 390, 485}
var ChainLightningLevel = [ChainLightningRanks + 1]int{0, 32, 40, 48, 56}

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
	baseDamage := ChainLightningBaseDamage[rank]
	spellCoeff := ChainLightningSpellCoef[rank]
	manaCost := ChainLightningManaCost[rank]
	level := ChainLightningLevel[rank]

	cooldown := time.Second * 6
	castTime := time.Millisecond * 2000

	shaman.ChainLightningBounceCoefficient = .70 // 30% reduction per bounce
	targetCount := ChainLightningTargetCount

	spell := shaman.newElectricSpellConfig(
		core.ActionID{SpellID: spellId},
		manaCost,
		castTime,
	)

	spell.SpellCode = SpellCode_ShamanChainLightning
	spell.RequiredLevel = level
	spell.Rank = rank
	spell.BonusCoefficient = spellCoeff
	spell.Cast.CD = core.Cooldown{
		Timer:    cdTimer,
		Duration: cooldown,
	}

	// Pool-sized ceiling, live-bounded loop; see APLActionMultidot.
	results := make([]*core.SpellResult, min(int(targetCount), len(shaman.Env.Encounter.AllTargetUnits)))

	spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		numHits := min(len(results), len(sim.Encounter.TargetUnits))

		// Shared by the primary bounce chain and Lightning Overload's
		// second one (talents.go's rollLightningOverload): the chain
		// starts from the primary target each time, and the damage
		// multiplier it entered with (halved for an overload) is
		// restored when the chain ends.
		dealBounces := func() {
			enteringMultiplier := spell.DamageMultiplier
			bounceTarget := target
			for hitIndex := 0; hitIndex < numHits; hitIndex++ {
				results[hitIndex] = spell.CalcDamage(sim, bounceTarget, baseDamage, spell.OutcomeMagicHitAndCrit)
				bounceTarget = sim.Environment.NextTargetUnit(bounceTarget)
				spell.DamageMultiplier *= shaman.ChainLightningBounceCoefficient
			}

			for _, result := range results[:numHits] {
				spell.DealDamage(sim, result)
			}

			spell.DamageMultiplier = enteringMultiplier
		}

		dealBounces()

		if shaman.rollLightningOverload(sim) {
			shaman.AtLightningOverloadScale(spell, dealBounces)
		}
	}

	return spell
}
