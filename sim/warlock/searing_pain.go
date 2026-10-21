package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const SearingPainRanks = 6

// baseDamage was the classic tooltip roll (rank 6 {208, 244}) until
// the rotation-accuracy audit compared it against spellconst/
// warlock.json's own per-rank flat "amount" (rank 6, 17923, amount 114,
// sp_coefficient 0.429 unchanged) - see shadowbolt.go's comment for
// the corroborating wowhead check and the same halving across the
// rest of the kit.
// SearingPainDamage is spellconst/warlock.json's own roll for ids 5676
// through 17923 (rank 6 rolls 107-125.8 at level 60: a centre of 114 at its
// own level, growing 1.2 a level) - see shadowbolt.go's comment on the
// classic roll it replaced.
var SearingPainDamage = [SearingPainRanks + 1]clientdamage.Effect{
	{},
	{Amount: 23, Variance: 0.210526, PerLevel: 0.6, SpellLevel: 18, MaxLevel: 24},
	{Amount: 33, Variance: 0.184615, PerLevel: 0.7, SpellLevel: 26, MaxLevel: 32},
	{Amount: 44, Variance: 0.189474, PerLevel: 0.8, SpellLevel: 34, MaxLevel: 40},
	{Amount: 62, Variance: 0.179104, PerLevel: 0.9, SpellLevel: 42, MaxLevel: 48},
	{Amount: 85, Variance: 0.17341, PerLevel: 1, SpellLevel: 50, MaxLevel: 56},
	{Amount: 114, Variance: 0.162162, PerLevel: 1.2, SpellLevel: 58, MaxLevel: 64},
}

func (warlock *Warlock) getSearingPainBaseConfig(rank int) core.SpellConfig {
	spellCoeff := [SearingPainRanks + 1]float64{0, .429, .429, .429, .429, .429, .429}[rank]
	damage := SearingPainDamage[rank]
	casterLevel := int(warlock.Level)
	spellId := [SearingPainRanks + 1]int32{0, 5676, 17919, 17920, 17921, 17922, 17923}[rank]
	manaCost := [SearingPainRanks + 1]float64{0, 45, 68, 91, 118, 141, 168}[rank]
	// Rank 3 (17920) is learned at level 34 in the client's own data
	// (1.60.1.70009), not 36.
	level := [SearingPainRanks + 1]int{0, 18, 26, 34, 42, 50, 58}[rank]
	castTime := time.Millisecond * 1500

	return core.SpellConfig{
		SpellCode:        SpellCode_WarlockSearingPain,
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		Flags:            core.SpellFlagAPL | core.SpellFlagResetAttackSwing | WarlockFlagDestruction,
		RequiredLevel:    level,
		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: castTime,
			},
		},
		// FOREVER: Improved Searing Pain is not in the client's trees.
		// BonusCritRating: 2.0 * float64(warlock.Talents.ImprovedSearingPain) * core.CritRatingPerCritChance,

		DamageMultiplier: 1,
		ThreatMultiplier: 2,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Decimation (talents/warlock.json 440870/440873): +3%/+6%
			// damage against a target below 35% health.
			oldMultiplier := spell.DamageMultiplier
			spell.DamageMultiplier *= warlock.decimationDamageMultiplier(target)
			spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
			spell.DamageMultiplier = oldMultiplier
		},
	}
}

func (warlock *Warlock) registerSearingPainSpell() {
	warlock.SearingPain = make([]*core.Spell, 0)
	for rank := 1; rank <= SearingPainRanks; rank++ {
		config := warlock.getSearingPainBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.SearingPain = append(warlock.SearingPain, warlock.GetOrRegisterSpell(config))
		}
	}
}
