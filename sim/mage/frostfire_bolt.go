package mage

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Frostfire Bolt: "Launches a bolt of frostfire at the enemy, causing
// 286 to 331 Frostfire damage, slowing movement speed by 40% and causing
// an additional 57 Frostfire damage over 9 sec. This spell will be
// checked against the lower of the target's Frost and Fire resists and
// counts as both Frost and Fire damage."
//
// The client has three trainable ranks: 401502 (level 40), 1237312
// (level 50) and 1237313 (level 60). Generated rank 0 (401735) is the
// level-1 "Gain the Frostfire Bolt ability" teaching spell and is never
// registered. The direct hit, cost, cast time and coefficient are the
// generated FrostfireBolt* arrays; the periodic effect (effect 2) is
// the hand table below.

// FrostfireBoltDotTickDamage is the client's per-tick amount of the
// periodic effect, flat with no growth per caster level and no spell
// power (effect 2's coefficient is 0): 9, 13 and 19 a tick are the
// tooltip's 27, 39 and 57 over 9 sec.
var FrostfireBoltDotTickDamage = [FrostfireBoltRanks + 1]float64{0, 9, 13, 19}

// FrostfireBoltDotTicks is duration_ms / period_ms, 9000 / 3000, the same
// on every rank (TestFrostfireBoltDotTicksFollowTheClientDuration).
const FrostfireBoltDotTicks int32 = 3

const frostfireBoltDotTickLength = 3 * time.Second

func (mage *Mage) registerFrostfireBoltSpell() {
	mage.FrostfireBolt = make([]*core.Spell, FrostfireBoltRanks+1)

	// Rank 0 is the teaching spell, so ranks start at 1.
	for rank := 1; rank <= FrostfireBoltRanks; rank++ {
		config := mage.newFrostfireBoltSpellConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.FrostfireBolt[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) newFrostfireBoltSpellConfig(rank int) core.SpellConfig {
	actionID := core.ActionID{SpellID: FrostfireBoltSpellId[rank]}
	roll := mage.clientRoll(FrostfireBoltBaseDamage[rank], FrostfireBoltPointsPerLevel[rank], FrostfireBoltLevel[rank], FrostfireBoltMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]
	baseDotDamage := FrostfireBoltDotTickDamage[rank]

	return core.SpellConfig{
		ActionID:       actionID,
		ClassSpellMask: MageSpellMaskFrostfireBolt,
		SpellCode:      SpellCode_MageFrostfireBolt,
		// Counts as both Frost and Fire: the school mods, Master of
		// Elements and the lower-of-two resist rule all read the mask.
		SpellSchool:      core.SpellSchoolFire | core.SpellSchoolFrost,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		// Binary like Frostbolt: the snare rides on the same hit roll,
		// so a resist takes the whole spell and no partial applies.
		Flags:        SpellFlagMage | SpellFlagChillSpell | core.SpellFlagBinary | core.SpellFlagAPL,
		MissileSpeed: 28,

		RequiredLevel: FrostfireBoltLevel[rank],
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: FrostfireBoltManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
				// Improved Fireball's reduction is a CastTime_Flat mod
				// in applyDeclarativeTalents, not an arithmetic term
				// here: one talent, one place.
				CastTime: time.Millisecond * time.Duration(FrostfireBoltCastTime[rank]),
			},
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:    fmt.Sprintf("Frostfire Bolt (Rank %d)", rank),
				ActionID: actionID.WithTag(1),
			},
			NumberOfTicks: FrostfireBoltDotTicks,
			TickLength:    frostfireBoltDotTickLength,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDotDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: FrostfireBoltSpellCoeff[rank],

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)

				if result.Landed() {
					spell.Dot(target).Apply(sim)
				}
			})
		},
	}
}
