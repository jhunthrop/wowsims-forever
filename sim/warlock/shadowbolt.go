package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const ShadowBoltRanks = 10

// ShadowBoltDamage is spellconst/warlock.json's own roll for ids 686
// through 25307 (rank 10 rolls 253.3-282.7 at level 60: a centre of 268 at
// level 60 growing 1.8 a level, 0.1067 wide), where the classic tooltip roll
// ({482, 538} at rank 10) was roughly twice that. Wowhead's Forever page
// for 25307 shows the same single "Value: 269 (SP mod: 0.857)". The same
// halving runs across the rest of the kit's nukes and DoTs, and the client
// wins.
var ShadowBoltDamage = [ShadowBoltRanks + 1]clientdamage.Effect{
	{},
	{Amount: 13, Variance: 0.285714, PerLevel: 0.2, SpellLevel: 1, MaxLevel: 5},
	{Amount: 23, Variance: 0.230769, PerLevel: 0.3, SpellLevel: 6, MaxLevel: 11},
	{Amount: 40, Variance: 0.153846, PerLevel: 0.6, SpellLevel: 12, MaxLevel: 17},
	{Amount: 58, Variance: 0.130435, PerLevel: 0.8, SpellLevel: 20, MaxLevel: 25},
	{Amount: 78, Variance: 0.131579, PerLevel: 1.2, SpellLevel: 28, MaxLevel: 33},
	{Amount: 101, Variance: 0.119816, PerLevel: 1.2, SpellLevel: 36, MaxLevel: 41},
	{Amount: 141, Variance: 0.114094, PerLevel: 1.4, SpellLevel: 44, MaxLevel: 49},
	{Amount: 191, Variance: 0.110236, PerLevel: 1.6, SpellLevel: 52, MaxLevel: 57},
	{Amount: 251, Variance: 0.108108, PerLevel: 1.8, SpellLevel: 60, MaxLevel: 65},
	{Amount: 268, Variance: 0.109804, PerLevel: 1.9, SpellLevel: 60, MaxLevel: 65},
}

func (warlock *Warlock) getShadowBoltBaseConfig(rank int) core.SpellConfig {
	spellCoeff := [ShadowBoltRanks + 1]float64{0, .486, .629, .8, .857, .857, .857, .857, .857, .857, .857}[rank]
	damage := ShadowBoltDamage[rank]
	casterLevel := int(warlock.Level)
	spellId := [ShadowBoltRanks + 1]int32{0, 686, 695, 705, 1088, 1106, 7641, 11659, 11660, 11661, 25307}[rank]
	manaCost := [ShadowBoltRanks + 1]float64{0, 25, 40, 70, 110, 160, 210, 265, 315, 370, 380}[rank]
	level := [ShadowBoltRanks + 1]int{0, 1, 6, 12, 20, 28, 36, 44, 52, 60, 60}[rank]
	castTime := [ShadowBoltRanks + 1]int32{0, 1700, 2200, 2800, 3000, 3000, 3000, 3000, 3000, 3000, 3000}[rank]

	return core.SpellConfig{
		SpellCode:        SpellCode_WarlockShadowBolt,
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolShadow,
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
			result := spell.CalcDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
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
