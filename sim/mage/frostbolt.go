package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Frostbolt keeps its hand-written per-rank arrays rather than taking
// the generated ones. The generator deliberately skips a name a package
// already declares ("skipped: \"Frostbolt\" already has a hand-written
// FrostboltRanks elsewhere in this package" in constants_auto_gen.go),
// and the two agree where it matters: rank 11 is spell 25304 at level
// 60, which TestFrostboltHasElevenRanks asserts. The client's rows stay
// resolvable through spellconst.Load for anything that needs the rest
// of the effect.
const FrostboltRanks = 11

var FrostboltSpellId = [FrostboltRanks + 1]int32{0, 116, 205, 837, 7322, 8406, 8407, 8408, 10179, 10180, 10181, 25304}
var FrostboltBaseDamage = [FrostboltRanks + 1][]float64{{0, 0}, {20, 22}, {33, 38}, {54, 61}, {78, 87}, {132, 144}, {180, 197}, {231, 251}, {301, 326}, {353, 383}, {440, 475}, {515, 555}}
var FrostboltSpellCoeff = [FrostboltRanks + 1]float64{0, .163, .269, .463, .706, .814, .814, .814, .814, .814, .814, .814}
var FrostboltCastTime = [FrostboltRanks + 1]int32{0, 1500, 1800, 2200, 2600, 3000, 3000, 3000, 3000, 3000, 3000, 3000}
var FrostboltManaCost = [FrostboltRanks + 1]float64{0, 25, 35, 50, 65, 100, 130, 160, 195, 225, 260, 290}
var FrostboltLevel = [FrostboltRanks + 1]int{0, 4, 8, 14, 20, 26, 32, 38, 44, 50, 56, 60}

func (mage *Mage) registerFrostboltSpell() {
	mage.Frostbolt = make([]*core.Spell, FrostboltRanks+1)

	maxRank := core.TernaryInt(core.IncludeAQ, FrostboltRanks, FrostboltRanks-1)
	for rank := 1; rank <= maxRank; rank++ {
		config := mage.getFrostboltConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.Frostbolt[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) getFrostboltConfig(rank int) core.SpellConfig {
	spellId := FrostboltSpellId[rank]
	baseDamageLow := FrostboltBaseDamage[rank][0]
	baseDamageHigh := FrostboltBaseDamage[rank][1]
	spellCoeff := FrostboltSpellCoeff[rank]
	castTime := FrostboltCastTime[rank]
	manaCost := FrostboltManaCost[rank]
	level := FrostboltLevel[rank]

	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: spellId},
		ClassSpellMask: MageSpellMaskFrostbolt,
		SpellCode:      SpellCode_MageFrostbolt,
		SpellSchool:    core.SpellSchoolFrost,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          SpellFlagMage | SpellFlagChillSpell | core.SpellFlagBinary | core.SpellFlagAPL,
		MissileSpeed:   28,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
				// Improved Frostbolt's reduction is a CastTime_Flat mod
				// in applyDeclarativeTalents, not an arithmetic term
				// here: one talent, one place.
				CastTime: time.Millisecond * time.Duration(castTime),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				if result.Landed() {
					spell.DealDamage(sim, result)
				}
			})
		},
	}
}
