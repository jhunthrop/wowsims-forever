package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (mage *Mage) registerBlastWaveSpell() {
	if !mage.Talents.BlastWave {
		return
	}

	mage.BlastWave = make([]*core.Spell, BlastWaveRanks+1)
	cdTimer := mage.NewTimer()

	for rank := 1; rank <= BlastWaveRanks; rank++ {
		config := mage.newBlastWaveSpellConfig(rank, cdTimer)

		if config.RequiredLevel <= int(mage.Level) {
			mage.BlastWave[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) newBlastWaveSpellConfig(rank int, cooldownTimer *core.Timer) core.SpellConfig {
	spellId := BlastWaveSpellId[rank]
	roll := mage.clientRoll(BlastWaveBaseDamage[rank], BlastWavePointsPerLevel[rank], BlastWaveLevel[rank], BlastWaveMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]
	manaCost := BlastWaveManaCost[rank]
	level := BlastWaveLevel[rank]

	spellCoeff := BlastWaveSpellCoeff[rank]
	cooldown := time.Second * 45

	return core.SpellConfig{
		SpellCode:        SpellCode_MageBlastWave,
		ClassSpellMask:   MageSpellMaskBlastWave,
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | core.SpellFlagBinary | core.SpellFlagAPL,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    cooldownTimer,
				Duration: cooldown,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
				spell.CalcAndDealDamage(sim, aoeTarget, baseDamage, spell.OutcomeMagicCrit)
			}
		},
	}
}
