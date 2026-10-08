package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Ice Lance: a fast, cheap Frost nuke that hits harder against a Frozen
// target. Vanilla has no such spell; Forever adds it as the Frost tree's
// tier-2 column-2 talent (node 105767), not as a baseline ability, so it
// is registered from ApplyTalents under the talent.
//
// Every number is the client's own, from sim/mage/constants_auto_gen.go
// (build 1.60.1.69893) rather than a Go literal, so a patch that
// renumbers a rank is a regeneration. Only the coefficient carries an
// unconfirmed marker, and the generated file carries it: the client's
// coefficient column was zero and the generator derived the value from
// the vanilla convention.
const (
	// "Deals 300% increased damage to Frozen targets."
	//
	// unconfirmed: whether Forever reads its own wording literally
	// (x4) or in Blizzard's usual sense of "triple damage" (x3). x3 is
	// what every previous implementation of this spell did, so it is
	// the value here, and the nightly cast-frequency and damage
	// comparison is what settles it. It applies against a Frost
	// Nova-frozen target and on a Fingers of Frost cast - see
	// isTargetFrozen.
	iceLanceFrozenMultiplier = 3.0
)

func (mage *Mage) registerIceLanceSpell() {
	if !mage.Talents.IceLance {
		return
	}

	mage.IceLance = make([]*core.Spell, IceLanceRanks+1)

	// Rank 0 of the client's Ice Lance rows is not a player rank: its
	// base-damage column carries the spell id 400640 rather than a
	// damage figure, which is the signature of a trigger or NPC variant.
	// Player ranks start at 1.
	for rank := 1; rank <= IceLanceRanks; rank++ {
		config := mage.getIceLanceConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.IceLance[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) getIceLanceConfig(rank int) core.SpellConfig {
	roll := mage.clientRoll(IceLanceBaseDamage[rank], IceLancePointsPerLevel[rank], IceLanceLevel[rank], IceLanceMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: IceLanceSpellId[rank]},
		ClassSpellMask:   MageSpellMaskIceLance,
		SpellSchool:      core.SpellSchoolFrost,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | core.SpellFlagAPL,
		// unconfirmed: the client's data carries no projectile speed;
		// 38 is the value every previous implementation used.
		MissileSpeed: 38,

		RequiredLevel: IceLanceLevel[rank],
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: IceLanceManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * time.Duration(IceLanceCastTime[rank]),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: IceLanceSpellCoeff[rank],

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
			// The Frozen bonus is a base-damage multiplier and Shatter's
			// is crit chance, so they compose without an interaction
			// rule.
			if mage.isTargetFrozen(target) {
				baseDamage *= iceLanceFrozenMultiplier
			}
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				if result.Landed() {
					spell.DealDamage(sim, result)
				}
			})
		},
	}
}

// isTargetFrozen reports whether target counts as Frozen for the cast
// that is resolving: it carries Frost Nova's Frozen aura, or the cast
// spent a Fingers of Frost charge (fingers_of_frost.go).
//
// Frostbite's Freeze (chill.go) is the third source: a root a raid boss
// is immune to, the way it is to Frost Nova's (canFreeze).
func (mage *Mage) isTargetFrozen(target *core.Unit) bool {
	return mage.fingersFreezeCast || mage.targetHasFrozenAura(target)
}

// targetHasFrozenAura is the target-side half of isTargetFrozen.
func (mage *Mage) targetHasFrozenAura(target *core.Unit) bool {
	return (mage.FrozenAuras != nil && mage.FrozenAuras.Get(target).IsActive()) ||
		(mage.FrostbiteFrozenAuras != nil && mage.FrostbiteFrozenAuras.Get(target).IsActive())
}
