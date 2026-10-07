package mage

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// PyroblastDotTickDamage is the client's per-tick amount of the
// periodic effect (effect 1 of each rank): flat, with no growth per caster
// level, and four ticks at three seconds.
var PyroblastDotTickDamage = [PyroblastRanks + 1]float64{0, 11, 14, 19, 25, 31, 38, 46, 53}

func (mage *Mage) registerPyroblastSpell() {
	if !mage.Talents.Pyroblast {
		return
	}

	mage.Pyroblast = make([]*core.Spell, PyroblastRanks+1)

	for rank := 1; rank <= PyroblastRanks; rank++ {
		config := mage.newPyroblastSpellConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.Pyroblast[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) newPyroblastSpellConfig(rank int) core.SpellConfig {

	numTicks := int32(4)
	tickLength := time.Second * 3

	spellId := PyroblastSpellId[rank]
	roll := mage.clientRoll(PyroblastBaseDamage[rank], PyroblastPointsPerLevel[rank], PyroblastLevel[rank], PyroblastMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]
	baseDotDamage := PyroblastDotTickDamage[rank]
	manaCost := PyroblastManaCost[rank]
	level := PyroblastLevel[rank]

	spellCoeff := PyroblastSpellCoeff[rank]
	dotCoeff := .15
	castTime := time.Second * 6

	actionID := core.ActionID{SpellID: spellId}

	spellConfig := core.SpellConfig{
		ActionID:         actionID,
		ClassSpellMask:   MageSpellMaskPyroblast,
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | core.SpellFlagAPL,
		MissileSpeed:     24,

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

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:    fmt.Sprintf("Pyroblast (Rank %d)", rank),
				ActionID: actionID.WithTag(1),
			},
			NumberOfTicks:    numTicks,
			TickLength:       tickLength,
			BonusCoefficient: dotCoeff,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDotDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)

				if result.Landed() {
					spell.Dot(target).Apply(sim)
				}
			})
		},
	}

	return spellConfig
}
