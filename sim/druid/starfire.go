package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const StarfireRanks = 7

var StarfireSpellId = [StarfireRanks + 1]int32{0, 2912, 8949, 8950, 8951, 9875, 9876, 25298}

// StarfireBaseDamage is spellconst/druid.json's flat per-rank "amount"
// (rank 7 is 381 at coefficient 1.0, against the Classic roll of 496-584
// it replaced); the coefficient was already the client's.
var StarfireDamage = [StarfireRanks + 1]clientdamage.Effect{
	{},
	{Amount: 81, Variance: 0.20202, PerLevel: 1.3, SpellLevel: 20, MaxLevel: 25},
	{Amount: 110, Variance: 0.197368, PerLevel: 1.5, SpellLevel: 26, MaxLevel: 32},
	{Amount: 144, Variance: 0.180995, PerLevel: 1.6, SpellLevel: 34, MaxLevel: 40},
	{Amount: 198, Variance: 0.175896, PerLevel: 1.8, SpellLevel: 42, MaxLevel: 48},
	{Amount: 266, Variance: 0.167089, PerLevel: 2.2, SpellLevel: 50, MaxLevel: 56},
	{Amount: 337, Variance: 0.164948, PerLevel: 2.3, SpellLevel: 58, MaxLevel: 64},
	{Amount: 381, Variance: 0.162963, PerLevel: 2.4, SpellLevel: 60, MaxLevel: 66},
}
var StarfireSpellCoeff = [StarfireRanks + 1]float64{0, 1, 1, 1, 1, 1, 1, 1}
var StarfireManaCost = [StarfireRanks + 1]float64{0, 95, 135, 180, 230, 275, 315, 340}
var StarfireLevel = [StarfireRanks + 1]int{0, 20, 26, 34, 42, 50, 58, 60}

func (druid *Druid) registerStarfireSpell() {
	druid.Starfire = make([]*DruidSpell, StarfireRanks+1)

	maxRank := core.TernaryInt(core.IncludeAQ, StarfireRanks, StarfireRanks-1)
	for rank := 1; rank <= maxRank; rank++ {
		config := druid.newStarfireSpellConfig(rank)

		if config.RequiredLevel <= int(druid.Level) {
			druid.Starfire[rank] = druid.RegisterSpell(Humanoid|Moonkin, config)
		}
	}
}

func (druid *Druid) newStarfireSpellConfig(rank int) core.SpellConfig {
	spellId := StarfireSpellId[rank]
	damage := StarfireDamage[rank]
	casterLevel := int(druid.Level)
	manaCost := StarfireManaCost[rank]
	level := StarfireLevel[rank]

	castTime := 3500

	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: spellId},
		SpellCode:      SpellCode_DruidStarfire,
		ClassSpellMask: DruidSpellMaskStarfire,
		SpellSchool:    core.SpellSchoolArcane,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          SpellFlagOmen | core.SpellFlagAPL | core.SpellFlagResetAttackSwing,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond*time.Duration(castTime) - time.Millisecond*100*time.Duration(druid.Talents.ImprovedStarfire),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: StarfireSpellCoeff[rank],
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
		},
	}
}
