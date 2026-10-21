package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// improvedScorchProcChance is Improved Scorch's chance by rank, Forever's
// 1.60.1.70291 talent text: "Your Scorch spell has a 33/67/100% chance to
// cause your target to be vulnerable to Fire damage".
var improvedScorchProcChance = [4]float64{0, 0.33, 0.67, 1}

func (mage *Mage) registerScorchSpell() {
	mage.Scorch = make([]*core.Spell, ScorchRanks+1)

	for rank := 1; rank <= ScorchRanks; rank++ {
		config := mage.getScorchConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.Scorch[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) getScorchConfig(rank int) core.SpellConfig {
	spellId := ScorchSpellId[rank]
	roll := mage.clientRoll(ScorchBaseDamage[rank], ScorchPointsPerLevel[rank], ScorchLevel[rank], ScorchMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]
	manaCost := ScorchManaCost[rank]
	level := ScorchLevel[rank]

	spellCoeff := ScorchSpellCoeff[rank]
	debuffProcChance := improvedScorchProcChance[rankIndex(mage.Talents.ImprovedScorch, improvedScorchProcChance[:])]

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		ClassSpellMask:   MageSpellMaskScorch,
		SpellCode:        SpellCode_MageScorch,
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            core.SpellFlagAPL | SpellFlagMage,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * 1500,
			},
		},

		// FOREVER: Incinerate is not in the client's trees.
		// BonusCritRating: 2 * float64(mage.Talents.Incinerate) * core.CritRatingPerCritChance,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			if sim.RandomFloat("Improved Scorch") < debuffProcChance {
				aura := mage.ImprovedScorchAuras.Get(target)
				aura.Activate(sim)
				aura.AddStack(sim)
			}
		},

		RelatedAuras: []core.AuraArray{mage.ImprovedScorchAuras},
	}
}
