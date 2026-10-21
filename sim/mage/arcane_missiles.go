package mage

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// The channel (spells 5143...25345) and the missile it fires each second
// (7268...25346, its trigger spell) are two client spells. The generated
// ArcaneMissiles* arrays in constants_auto_gen.go are the missile's: the
// per-missile damage roll, its growth per level, its level cap and its
// coefficient. The ArcaneMissilesChannel* tables are the channel's own
// ids, cast time in seconds (one missile per second), cost and level.
var ArcaneMissilesChannelSpellId = [ArcaneMissilesRanks + 1]int32{0, 5143, 5144, 5145, 8416, 8417, 10211, 10212, 25345}
var ArcaneMissilesChannelCastTime = [ArcaneMissilesRanks + 1]int32{0, 3, 4, 5, 5, 5, 5, 5, 5}
var ArcaneMissilesChannelManaCost = [ArcaneMissilesRanks + 1]float64{0, 85, 140, 235, 320, 410, 500, 595, 655}
var ArcaneMissilesChannelLevel = [ArcaneMissilesRanks + 1]int{0, 8, 16, 24, 32, 40, 48, 56, 56}

func (mage *Mage) registerArcaneMissilesSpell() {
	mage.ArcaneMissiles = make([]*core.Spell, ArcaneMissilesRanks+1)
	mage.ArcaneMissilesTickSpell = make([]*core.Spell, ArcaneMissilesRanks+1)

	// TODO AQ <=
	for rank := 1; rank <= ArcaneMissilesRanks; rank++ {
		config := mage.getArcaneMissilesSpellConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.ArcaneMissiles[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) getArcaneMissilesSpellConfig(rank int) core.SpellConfig {
	spellId := ArcaneMissilesChannelSpellId[rank]
	castTime := ArcaneMissilesChannelCastTime[rank]
	manaCost := ArcaneMissilesChannelManaCost[rank]
	level := ArcaneMissilesChannelLevel[rank]
	missile := mage.arcaneMissileRoll(rank)
	baseTickDamage := (missile[0] + missile[1]) / 2

	numTicks := castTime
	tickLength := time.Second

	tickSpell := mage.getArcaneMissilesTickSpell(rank)
	mage.ArcaneMissilesTickSpell[rank] = tickSpell

	return core.SpellConfig{
		SpellCode:      SpellCode_MageArcaneMissiles,
		ClassSpellMask: MageSpellMaskArcaneMissiles,
		ActionID:       core.ActionID{SpellID: spellId},
		SpellSchool:    core.SpellSchoolArcane,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		// The channel carries the cast count and, with its tick a passive
		// spell, the damage too; without metrics here the ladder reads
		// Arcane Missiles as never cast.
		Flags: SpellFlagMage | core.SpellFlagAPL | core.SpellFlagChanneled,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: fmt.Sprintf("ArcaneMissiles-%d-%d", +rank, numTicks),
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					// TODO: This check is necessary to ensure the final tick occurs before
					// Arcane Blast stacks are dropped. To fix this, ticks need to reliably
					// occur before aura expirations.

					//TODO: Test interaction in classic code without aura
					dot := mage.ArcaneMissiles[rank].Dot(aura.Unit)
					if dot.TickCount < dot.NumberOfTicks {
						dot.TickCount++
						dot.TickOnce(sim)
					}
				},
			},
			NumberOfTicks: numTicks,
			TickLength:    tickLength,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				tickSpell.Cast(sim, target)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			dot := spell.Dot(target)
			// Missile Barrage (missile_barrage.go): "missiles fire every
			// 0.5 sec" while its buff is up. The buff's own Cost.Multiplier
			// swap is what makes this particular cast free; checked here
			// too because OnCastComplete (which consumes the buff) does
			// not run until after this function returns.
			if mage.MissileBarrageAura != nil && mage.MissileBarrageAura.IsActive() {
				dot.TickLength = missileBarrageTickLength
			} else {
				dot.TickLength = tickLength
			}
			dot.Apply(sim)
		},
		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			return tickSpell.CalcDamage(sim, target, baseTickDamage, spell.OutcomeExpectedMagicHitAndCrit)
		},
	}
}

// arcaneMissileRoll is the client's roll for one missile of a rank at
// this mage's level; the missile's own spell level, not the channel's,
// sets where its growth per level starts.
func (mage *Mage) arcaneMissileRoll(rank int) [2]float64 {
	return mage.clientRoll(ArcaneMissilesBaseDamage[rank], ArcaneMissilesPointsPerLevel[rank], ArcaneMissilesLevel[rank], ArcaneMissilesMaxLevel[rank])
}

func (mage *Mage) getArcaneMissilesTickSpell(rank int) *core.Spell {
	spellId := ArcaneMissilesChannelSpellId[rank]
	missile := mage.arcaneMissileRoll(rank)
	spellCoeff := ArcaneMissilesSpellCoeff[rank]

	return mage.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_MageArcaneMissilesTick,
		ClassSpellMask: MageSpellMaskArcaneMissilesTick,
		ActionID:       core.ActionID{SpellID: spellId}.WithTag(1),
		SpellSchool:    core.SpellSchoolArcane,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          SpellFlagMage | core.SpellFlagPassiveSpell,
		MissileSpeed:   20,

		Rank: 1,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: missile,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, sim.Roll(missile[0], missile[1]), spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
