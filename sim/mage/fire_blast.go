package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const FireBlastRanks = 7

var FireBlastSpellId = [FireBlastRanks + 1]int32{0, 2136, 2137, 2138, 8412, 8413, 10197, 10199}
var FireBlastBaseDamage = [FireBlastRanks + 1][]float64{{0, 0}, {24, 32}, {51.6562, 64.3438}, {89.5652, 110.4348}, {148.0216, 177.9784}, {214.7068, 257.2932}, {302.7328, 359.2672}, {401.6553, 474.3447}}
var FireBlastPointsPerLevel = [FireBlastRanks + 1]float64{0, 0.6, 1, 1.4, 1.8, 2.2, 2.6, 3}
var FireBlastMaxLevel = [FireBlastRanks + 1]int{0, 11, 19, 27, 35, 43, 51, 59}
var FireBlastSpellCoeff = [FireBlastRanks + 1]float64{0, 0.429, 0.429, 0.429, 0.429, 0.429, 0.429, 0.429}
var FireBlastManaCost = [FireBlastRanks + 1]float64{0, 40, 75, 115, 165, 220, 280, 340}
var FireBlastLevel = [FireBlastRanks + 1]int{0, 6, 14, 22, 30, 38, 46, 54}

func (mage *Mage) registerFireBlastSpell() {
	mage.FireBlast = make([]*core.Spell, FireBlastRanks+1)
	cdTimer := mage.NewTimer()

	for rank := 1; rank <= FireBlastRanks; rank++ {
		config := mage.newFireBlastSpellConfig(rank, cdTimer)

		if config.RequiredLevel <= int(mage.Level) {
			mage.FireBlast[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) newFireBlastSpellConfig(rank int, cdTimer *core.Timer) core.SpellConfig {

	spellId := FireBlastSpellId[rank]
	roll := mage.clientRoll(FireBlastBaseDamage[rank], FireBlastPointsPerLevel[rank], FireBlastLevel[rank], FireBlastMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]
	spellCoeff := FireBlastSpellCoeff[rank]
	manaCost := FireBlastManaCost[rank]
	level := FireBlastLevel[rank]

	cooldown := time.Second * 8
	flags := SpellFlagMage | core.SpellFlagAPL

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		ClassSpellMask:   MageSpellMaskFireBlast,
		SpellCode:        SpellCode_MageFireBlast,
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            flags,

		Rank:          rank,
		RequiredLevel: level,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer: cdTimer,
				// FOREVER: Improved Fire Blast is not in the client's trees.
				// Duration: cooldown - time.Millisecond*500*time.Duration(mage.Talents.ImprovedFireBlast),
				Duration: cooldown,
			},
		},

		// FOREVER: Incinerate is not in the client's trees.
		// BonusCritRating: 2 * float64(mage.Talents.Incinerate) * core.CritRatingPerCritChance,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			baseDamageCalc := (baseDamageLow + baseDamageHigh) / 2
			return spell.CalcDamage(sim, target, baseDamageCalc, spell.OutcomeExpectedMagicHitAndCrit)
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
		},
	}
}
