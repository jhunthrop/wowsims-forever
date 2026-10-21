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
	roll := mage.clientRoll(FrostboltBaseDamage[rank], FrostboltPointsPerLevel[rank], FrostboltLevel[rank], FrostboltMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]
	spellCoeff := FrostboltSpellCoeff[rank]
	castTime := FrostboltCastTime[rank]
	manaCost := FrostboltManaCost[rank]
	level := FrostboltLevel[rank]

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		ClassSpellMask:   MageSpellMaskFrostbolt,
		SpellCode:        SpellCode_MageFrostbolt,
		SpellSchool:      core.SpellSchoolFrost,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | SpellFlagChillSpell | core.SpellFlagBinary | core.SpellFlagAPL,
		MissileSpeed:     28,

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
