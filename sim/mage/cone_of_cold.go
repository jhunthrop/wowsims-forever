package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Cone of Cold is a baseline Frost ability every mage trains (Frost skill
// line, SkillLineAbility), so it registers from Initialize like Frost
// Nova. Client rows (build 1.60.1.70009), ranks 120, 8492, 10159, 10160
// and 10161 at levels 26/34/42/50/58:
//   - instant (cast index 1), 10 s category cooldown, 1.5 s global,
//     210..555 mana (constants_auto_gen.go's ConeOfCold* arrays);
//   - effect 0: aura 33, a 40% movement snare for 6 s (duration index 32);
//   - effect 1: school damage with coefficient 0.129, hitting every target
//     in the cone (target type 24, radius index 13);
//   - SpellClassMask_0 1573376 carries the Chill bit (1048576), so a
//     landed Cone of Cold is a Chill effect for Fingers of Frost and
//     Frostbite (chill.go).
//
// Rank 0 of the generated arrays (30095) is a creature-ability variant
// and is not a player rank.
//
// Model decisions the rows do not carry:
//   - Binary: the spell is a damage effect plus a snare aura, the same
//     structure as Frostbolt, which the engine treats as binary.
//   - Every target of the encounter stands in the cone; the snare itself
//     changes nothing on a stationary target.
func (mage *Mage) registerConeOfColdSpell() {
	mage.ConeOfCold = make([]*core.Spell, ConeOfColdRanks+1)

	cdTimer := mage.NewTimer()
	for rank := 1; rank <= ConeOfColdRanks; rank++ {
		config := mage.getConeOfColdConfig(rank, cdTimer)

		if config.RequiredLevel <= int(mage.Level) {
			mage.ConeOfCold[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) getConeOfColdConfig(rank int, cdTimer *core.Timer) core.SpellConfig {
	roll := mage.clientRoll(ConeOfColdBaseDamage[rank], ConeOfColdPointsPerLevel[rank], ConeOfColdLevel[rank], ConeOfColdMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: ConeOfColdSpellId[rank]},
		ClassSpellMask:   MageSpellMaskConeOfCold,
		SpellCode:        SpellCode_MageConeOfCold,
		SpellSchool:      core.SpellSchoolFrost,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | SpellFlagChillSpell | core.SpellFlagBinary | core.SpellFlagAPL,

		RequiredLevel: ConeOfColdLevel[rank],
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: ConeOfColdManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: time.Duration(ConeOfColdCooldownMS[rank]) * time.Millisecond,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: ConeOfColdSpellCoeff[rank],

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
				spell.CalcAndDealDamage(sim, aoeTarget, baseDamage, spell.OutcomeMagicHitAndCrit)
			}
		},
	}
}
