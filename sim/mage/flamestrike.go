package mage

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const FlamestrikeRanks = 6

var FlamestrikeSpellId = [FlamestrikeRanks + 1]int32{0, 2120, 2121, 8422, 8423, 10215, 10216}
var FlamestrikeBaseDamage = [FlamestrikeRanks + 1][]float64{{0, 0}, {52, 68}, {96, 122}, {154, 192}, {220, 272}, {291, 359}, {375, 459}}
var FlamestrikePointsPerLevel = [FlamestrikeRanks + 1]float64{0, 0.6, 0.8, 1, 1.3, 1.5, 1.7}
var FlamestrikeMaxLevel = [FlamestrikeRanks + 1]int{0, 21, 29, 37, 45, 53, 61}
var FlamestrikeSpellCoeff = [FlamestrikeRanks + 1]float64{0, 0.157, 0.157, 0.157, 0.157, 0.157, 0.157}

// FlamestrikeDotTickDamage is the client's per-tick amount of the ground
// aura's own rows (1279983...1279990): flat, no growth with level, and not
// the direct effect's - the dot has its own amount and coefficient.
var FlamestrikeDotTickDamage = [FlamestrikeRanks + 1]float64{0, 11, 21, 33, 47, 64, 83}
var FlamestrikeDotSpellCoeff = [FlamestrikeRanks + 1]float64{0, 0.032, 0.032, 0.032, 0.032, 0.032, 0.032}
var FlamestrikeManaCost = [FlamestrikeRanks + 1]float64{0, 195, 330, 490, 650, 815, 990}
var FlamestrikeLevel = [FlamestrikeRanks + 1]int{0, 16, 24, 32, 40, 48, 56}

func (mage *Mage) registerFlamestrikeSpell() {
	mage.Flamestrike = make([]*core.Spell, FlamestrikeRanks+1)

	for rank := 1; rank <= FlamestrikeRanks; rank++ {
		config := mage.newFlamestrikeSpellConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.Flamestrike[rank] = mage.GetOrRegisterSpell(config)
			// See blizzard.go's identical wiring: the AOE dot's Aura
			// lives on the caster, not a target, so it is wired as a
			// RelatedSelfBuff post-registration for the conformance
			// report to read.
			mage.Flamestrike[rank].RelatedSelfBuff = mage.Flamestrike[rank].AOEDot().Aura
		}
	}
}

func (mage *Mage) newFlamestrikeSpellConfig(rank int) core.SpellConfig {
	numTicks := int32(4)
	tickLength := time.Second * 2

	spellId := FlamestrikeSpellId[rank]
	roll := mage.clientRoll(FlamestrikeBaseDamage[rank], FlamestrikePointsPerLevel[rank], FlamestrikeLevel[rank], FlamestrikeMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]
	baseDotDamage := FlamestrikeDotTickDamage[rank]
	spellCoeff := FlamestrikeSpellCoeff[rank]
	dotCoeff := FlamestrikeDotSpellCoeff[rank]
	manaCost := FlamestrikeManaCost[rank]
	level := FlamestrikeLevel[rank]

	castTime := time.Second * 3

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		ClassSpellMask:   MageSpellMaskFlamestrike,
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | core.SpellFlagAPL,
		SpellCode:        SpellCode_MageFlamestrike,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: castTime,
			},
		},

		BonusCritRating: float64(5 * mage.Talents.ImprovedFlamestrike * core.CritRatingPerCritChance),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				Label: fmt.Sprintf("Flamestrike (Rank %d)", rank),
			},
			NumberOfTicks:    numTicks,
			TickLength:       tickLength,
			BonusCoefficient: dotCoeff,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDotDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				for _, aoeTarget := range sim.Encounter.TargetUnits {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, aoeTarget, dot.OutcomeTick)
				}
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
				spell.CalcAndDealDamage(sim, aoeTarget, baseDamage, spell.OutcomeMagicCrit)
			}
			spell.AOEDot().Apply(sim)
		},
	}
}
