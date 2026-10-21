package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const WrathRanks = 8

var WrathSpellId = [WrathRanks + 1]int32{0, 5176, 5177, 5178, 5179, 5180, 6780, 8905, 9912}

// WrathBaseDamage and WrathSpellCoeff are spellconst/druid.json's flat
// per-rank "amount" and "sp_coefficient" for ids 5176-9912 (rank 8 is 91
// at 0.571, where the Classic roll of 248-277 it replaced was nearly three
// times that). sim/druid/spellconst_damage_test.go checks every rank
// against the vendored client file.
// WrathDamage is spellconst/druid.json's own roll for ids 5176 through
// 9912: the centre at the spell's level, the per-level growth to the cap
// and the width of the roll (rank 8 rolls 91.6-102.4 at level 60, a
// centre of 97 on 91 at level 54).
var WrathDamage = [WrathRanks + 1]clientdamage.Effect{
	{},
	{Amount: 15, Variance: 0.153846, PerLevel: 0.2, SpellLevel: 1, MaxLevel: 5},
	{Amount: 21, Variance: 0.148148, PerLevel: 0.3, SpellLevel: 6, MaxLevel: 12},
	{Amount: 30, Variance: 0.166667, PerLevel: 0.5, SpellLevel: 14, MaxLevel: 20},
	{Amount: 37, Variance: 0.147059, PerLevel: 0.6, SpellLevel: 22, MaxLevel: 28},
	{Amount: 45, Variance: 0.12963, PerLevel: 0.7, SpellLevel: 30, MaxLevel: 36},
	{Amount: 54, Variance: 0.121622, PerLevel: 0.9, SpellLevel: 38, MaxLevel: 44},
	{Amount: 68, Variance: 0.110553, PerLevel: 0.9, SpellLevel: 46, MaxLevel: 52},
	{Amount: 91, Variance: 0.112, PerLevel: 1, SpellLevel: 54, MaxLevel: 60},
}
var WrathSpellCoeff = [WrathRanks + 1]float64{0, 0.429, 0.486, 0.571, 0.571, 0.571, 0.571, 0.571, 0.571}

// WrathManaCost was a stale pre-Forever table (roughly 40-75% above the
// client's real per-rank cost at every rank); corrected against
// 1.60.1.70009 spellconst/druid.json (ids 5176-9912). The generator
// skips Wrath's own name ("skipped: \"Wrath\" already has a
// hand-written WrathRanks elsewhere in this package" in
// constants_auto_gen.go), so this hand table is the source of truth
// and must be kept in sync by hand, the same situation as Frostbolt's
// in sim/mage/frostbolt.go.
var WrathManaCost = [WrathRanks + 1]float64{0, 10, 20, 40, 50, 70, 80, 100, 120}
var WrathCastTime = [WrathRanks + 1]int{0, 1500, 1700, 2000, 2000, 2000, 2000, 2000, 2000}
var WrathLevel = [WrathRanks + 1]int{0, 1, 6, 14, 22, 30, 38, 46, 54}

func (druid *Druid) registerWrathSpell() {
	druid.Wrath = make([]*DruidSpell, WrathRanks+1)

	for rank := 1; rank <= WrathRanks; rank++ {
		config := druid.newWrathSpellConfig(rank)

		if config.RequiredLevel <= int(druid.Level) {
			druid.Wrath[rank] = druid.RegisterSpell(Humanoid|Moonkin, config)
		}
	}
}

func (druid *Druid) newWrathSpellConfig(rank int) core.SpellConfig {
	spellId := WrathSpellId[rank]
	damage := WrathDamage[rank]
	casterLevel := int(druid.Level)
	spellCoeff := WrathSpellCoeff[rank]
	manaCost := WrathManaCost[rank]
	castTime := WrathCastTime[rank]
	level := WrathLevel[rank]

	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: spellId},
		SpellCode:      SpellCode_DruidWrath,
		ClassSpellMask: DruidSpellMaskWrath,
		SpellSchool:    core.SpellSchoolNature,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          SpellFlagOmen | core.SpellFlagAPL | core.SpellFlagResetAttackSwing,

		RequiredLevel: level,
		Rank:          rank,
		MissileSpeed:  20,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond*time.Duration(castTime) - time.Millisecond*100*time.Duration(druid.Talents.ImprovedWrath),
			},
		},

		DamageMultiplier: 1, // + core.Ternary(druid.Ranged().ID == IdolOfWrath, .02, 0),
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)

			// NG procs when the cast finishes
			if result.DidCrit() && druid.NaturesGraceProcAura != nil {
				druid.NaturesGraceProcAura.Activate(sim)
			}

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	}
}
