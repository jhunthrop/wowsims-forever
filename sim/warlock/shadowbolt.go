package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const ShadowBoltRanks = 10

// baseDamage was the classic tooltip roll (e.g. rank 10 {482, 538})
// until the rotation-accuracy audit compared it against spellconst/
// warlock.json's own per-rank "amount": every rank here is a single
// flat scalar (rank 10 amount 268, sp_coefficient 0.857 - unchanged),
// roughly half the classic roll's average, and corroborated by
// wowhead's Forever page for 25307 showing a single "Value: 269 (SP
// mod: 0.857)" with no min-max tooltip at all. The same halving shows
// up across every other warlock nuke that still carried a classic
// roll (Searing Pain, Soul Fire, Shadowburn, Conflagrate's classic
// ranks, Imp's Firebolt) and the DoTs (Corruption, Bane of Agony,
// Immolate) - Forever's client states a single number here the same
// way it does for Wrack and Incinerate, and the client wins.
func (warlock *Warlock) getShadowBoltBaseConfig(rank int) core.SpellConfig {
	spellCoeff := [ShadowBoltRanks + 1]float64{0, .14, .299, .56, .857, .857, .857, .857, .857, .857, .857}[rank]
	baseDamage := [ShadowBoltRanks + 1]float64{0, 13, 25, 41, 56, 78, 101, 141, 191, 251, 268}[rank]
	spellId := [ShadowBoltRanks + 1]int32{0, 686, 695, 705, 1088, 1106, 7641, 11659, 11660, 11661, 25307}[rank]
	manaCost := [ShadowBoltRanks + 1]float64{0, 25, 40, 70, 110, 160, 210, 265, 315, 370, 380}[rank]
	level := [ShadowBoltRanks + 1]int{0, 1, 6, 12, 20, 28, 36, 44, 52, 60, 60}[rank]
	castTime := [ShadowBoltRanks + 1]int32{0, 1700, 2200, 2800, 3000, 3000, 3000, 3000, 3000, 3000, 3000}[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_WarlockShadowBolt,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolShadow,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | core.SpellFlagResetAttackSwing | WarlockFlagDestruction,
		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * time.Duration(castTime),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Decimation (talents/warlock.json 440870/440873): +3%/+6%
			// damage against a target below 35% health.
			oldMultiplier := spell.DamageMultiplier
			spell.DamageMultiplier *= warlock.decimationDamageMultiplier(target)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.DamageMultiplier = oldMultiplier
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	}
}

func (warlock *Warlock) registerShadowBoltSpell() {
	warlock.ShadowBolt = make([]*core.Spell, 0)

	maxRank := core.TernaryInt(core.IncludeAQ, ShadowBoltRanks, ShadowBoltRanks-1)
	for rank := 1; rank <= maxRank; rank++ {
		config := warlock.getShadowBoltBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.ShadowBolt = append(warlock.ShadowBolt, warlock.GetOrRegisterSpell(config))
		}
	}
}
