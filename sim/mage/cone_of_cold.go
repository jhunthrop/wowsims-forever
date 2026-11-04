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
//   - Every target of the encounter stands in the cone when the mage does;
//     the snare itself changes nothing on a stationary target.
//   - Reach: the cone is radius index 13, 10 yards, widened 10% a rank by
//     Arctic Reach ("the radius of your Frost Nova and Cone of Cold spells
//     by 10/20%"). A mage standing farther from the target than that
//     (unit.DistanceFromTarget, the site's casters stand at 30) cannot
//     cast it at the target, so a rotation line for it is skipped.
func (mage *Mage) registerConeOfColdSpell() {
	mage.ConeOfCold = make([]*core.Spell, ConeOfColdRanks+1)

	cdTimer := mage.NewTimer()
	reach := mage.coneOfColdReach()
	for rank := 1; rank <= ConeOfColdRanks; rank++ {
		config := mage.getConeOfColdConfig(rank, cdTimer)
		config.ExtraCastCondition = func(_ *core.Simulation, _ *core.Unit) bool {
			return mage.DistanceFromTarget <= reach
		}

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
				// Winter's Chill (talents.go): the stacks this mage put on
				// the target raise its damage taken from this Cone of Cold
				// 20% a stack and add 7% of spell damage a stack, and the
				// cast spends them ("your next Cone of Cold"). A debuff of
				// its own, so the 20% multiplies with Improved Cone of Cold
				// rather than adding to it. unconfirmed: a resisted Cone of
				// Cold is taken to spend the stacks too.
				stacks := float64(mage.wintersChillStacks(aoeTarget))
				multiplier := 1 + wintersChillConeOfColdDamagePerStack*stacks
				coefficient := mage.wintersChillConeOfColdCoefficient(spell, stacks)
				spell.DamageMultiplier *= multiplier
				spell.BonusCoefficient += coefficient
				spell.CalcAndDealDamage(sim, aoeTarget, baseDamage, spell.OutcomeMagicHitAndCrit)
				spell.BonusCoefficient -= coefficient
				spell.DamageMultiplier /= multiplier
				mage.spendWintersChill(sim, aoeTarget)
			}
		},
	}
}

// coneOfColdRadius is the client's radius index 13; arcticReachRadiusPerRank
// is Arctic Reach's 10% a rank.
const (
	coneOfColdRadius         = 10.0
	arcticReachRadiusPerRank = 0.10
)

// coneOfColdReach is how far from the target the mage can stand and still
// catch it in the cone.
func (mage *Mage) coneOfColdReach() float64 {
	return coneOfColdRadius * (1 + arcticReachRadiusPerRank*float64(mage.Talents.ArcticReach))
}

// Winter's Chill on Cone of Cold (client 1.60.1.70707 talent text,
// "increasing the target's damage taken from your next Cone of Cold by
// 20% and it gains an additional 7% of your spell damage, stacking up to
// 5 times"): a stack is 20% more damage taken from the mage's next Cone
// of Cold (5% through 70685) and 7% of spell damage added to it. The
// shipped SpellEffect rows on 11180 carry no 20; the text is the hotfix.
const (
	wintersChillConeOfColdDamagePerStack      = 0.20
	wintersChillConeOfColdCoefficientPerStack = 0.07
)

// wintersChillConeOfColdCoefficient is the spell power coefficient stacks
// Winter's Chill stacks add to cone. It is the cast rank's spell power, so a
// rank far below the caster's level keeps only its level-gap share of it
// (core.LevelPenalty), as the rank's own coefficient does.
func (mage *Mage) wintersChillConeOfColdCoefficient(cone *core.Spell, stacks float64) float64 {
	return wintersChillConeOfColdCoefficientPerStack * stacks * core.LevelPenalty(cone.RequiredLevel, int(mage.Level))
}

// wintersChillStacks is the Winter's Chill stacks this mage holds on
// target.
func (mage *Mage) wintersChillStacks(target *core.Unit) int32 {
	if mage.WintersChillAuras == nil {
		return 0
	}
	aura := mage.WintersChillAuras.Get(target)
	if aura == nil || !aura.IsActive() {
		return 0
	}
	return aura.GetStacks()
}

// spendWintersChill is a Cone of Cold consuming the stacks on target.
func (mage *Mage) spendWintersChill(sim *core.Simulation, target *core.Unit) {
	if mage.wintersChillStacks(target) == 0 {
		return
	}
	mage.WintersChillAuras.Get(target).Deactivate(sim)
}
