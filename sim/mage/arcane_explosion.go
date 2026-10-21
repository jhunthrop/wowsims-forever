package mage

import (
	"github.com/wowsims/classic/sim/core"
)

func (mage *Mage) registerArcaneExplosionSpell() {
	mage.ArcaneExplosion = make([]*core.Spell, ArcaneExplosionRanks+1)

	for rank := 1; rank <= ArcaneExplosionRanks; rank++ {
		config := mage.newArcaneExplosionSpellConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.ArcaneExplosion[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) newArcaneExplosionSpellConfig(rank int) core.SpellConfig {
	spellId := ArcaneExplosionSpellId[rank]
	roll := mage.clientRoll(ArcaneExplosionBaseDamage[rank], ArcaneExplosionPointsPerLevel[rank], ArcaneExplosionLevel[rank], ArcaneExplosionMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]
	spellCoeff := ArcaneExplosionSpellCoeff[rank]
	manaCost := ArcaneExplosionManaCost[rank]
	level := ArcaneExplosionLevel[rank]

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		ClassSpellMask:   MageSpellMaskArcaneExplosion,
		SpellCode:        SpellCode_MageArcaneExplosion,
		SpellSchool:      core.SpellSchoolArcane,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | core.SpellFlagAPL,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				damage := sim.Roll(baseDamageLow, baseDamageHigh)
				spell.CalcAndDealDamage(sim, aoeTarget, damage, spell.OutcomeMagicCrit)
			}
		},
	}
}
