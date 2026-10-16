package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Arcane Wrath is the Arcane tree's tier-3 bool talent (node 113627, one
// rank, new in client 1.60.1.70425, behind Arcane Blast): "Launches a
// projectile of pure Arcane energy at the enemy target, causing N Arcane
// damage. This spell gains additional Spell Damage equal to 5% of your
// maximum Mana." Four learnable ranks (ArcaneWrath* in
// constants_auto_gen.go): a 3.5 sec cast on a 6 sec cooldown, coefficient
// 1.0, projectile speed 24 (SpellMisc.Speed).
//
// Like Arcane Blast and Ice Lance it is registered from ApplyTalents, so
// a mage that never spent the point has no spell to cast.
const (
	// "additional Spell Damage equal to 5% of your maximum Mana"
	// (SpellEffect 1363790, effect 2, base points 5).
	arcaneWrathManaShare    = 0.05
	arcaneWrathMissileSpeed = 24
)

func (mage *Mage) registerArcaneWrathSpell() {
	if !mage.Talents.ArcaneWrath {
		return
	}

	mage.ArcaneWrath = make([]*core.Spell, ArcaneWrathRanks+1)
	cdTimer := mage.NewTimer()

	for rank := 1; rank <= ArcaneWrathRanks; rank++ {
		config := mage.getArcaneWrathConfig(rank, cdTimer)

		if config.RequiredLevel <= int(mage.Level) {
			mage.ArcaneWrath[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) getArcaneWrathConfig(rank int, cdTimer *core.Timer) core.SpellConfig {
	roll := mage.clientRoll(ArcaneWrathBaseDamage[rank], ArcaneWrathPointsPerLevel[rank], ArcaneWrathLevel[rank], ArcaneWrathMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: ArcaneWrathSpellId[rank]},
		ClassSpellMask:   MageSpellMaskArcaneWrath,
		SpellCode:        SpellCode_MageArcaneWrath,
		SpellSchool:      core.SpellSchoolArcane,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | core.SpellFlagAPL,
		MissileSpeed:     arcaneWrathMissileSpeed,

		RequiredLevel: ArcaneWrathLevel[rank],
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: ArcaneWrathManaCost[rank],
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Duration(ArcaneWrathCastTime[rank]) * time.Millisecond,
			},
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: time.Duration(ArcaneWrathCooldownMS[rank]) * time.Millisecond,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: ArcaneWrathSpellCoeff[rank],

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// The mana share is extra spell damage for this cast, so it
			// takes the spell's own coefficient like any other spell power.
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh) +
				arcaneWrathManaShare*mage.MaxMana()*spell.BonusCoefficient
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				if result.Landed() {
					spell.DealDamage(sim, result)
				}
			})
		},
	}
}
