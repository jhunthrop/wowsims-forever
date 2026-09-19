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
	// comparison is what settles it. The figure is inert until
	// something in this sim can freeze a target - see isTargetFrozen.
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
	baseDamageLow := IceLanceBaseDamage[rank][0]
	baseDamageHigh := IceLanceBaseDamage[rank][1]

	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: IceLanceSpellId[rank]},
		ClassSpellMask: MageSpellMaskIceLance,
		SpellSchool:    core.SpellSchoolFrost,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          SpellFlagMage | core.SpellFlagAPL,
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
			// Whether the Frozen bonus stacks with Shatter's crit bonus
			// is exactly the kind of interaction rule the client's data
			// does not carry (research/07-simulator.md 5.3), so it is
			// written as multiplicative and marked.
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

// isTargetFrozen reports whether a Frost snare or root this mage applied
// is on the target.
//
// It is always false today, and that is the honest answer rather than a
// missing feature: this package registers neither Frost Nova nor
// Frostbite, so nothing here applies a Freeze, and a raid boss is immune
// to both in any case. Ice Lance's Frozen bonus, Shatter and Fingers of
// Frost therefore all read the same false, from this one place. When a
// Freeze aura exists, it is added to frozenAuras and all three start
// working together.
func (mage *Mage) isTargetFrozen(target *core.Unit) bool {
	for _, aura := range mage.frozenAuras(target) {
		if aura.IsActive() {
			return true
		}
	}
	return false
}

// frozenAuras is the set of auras that mean "this target is Frozen".
// Empty until a Freeze effect is registered; see isTargetFrozen.
func (mage *Mage) frozenAuras(_ *core.Unit) []*core.Aura {
	return nil
}
