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
// renumbers a rank is a regeneration. The coefficient is the exception:
// the client's column is zero, the generated file's value is only the
// vanilla convention, and iceLanceSpellCoeff takes it from Forever logs.
const (
	// "Deals 300% increased damage to Frozen targets." (1.60.1.70707
	// talent text; 70425 through 70685 said 200%, 70374 said 300%). The
	// shipped SpellEffect row 1342606 (spell 1312002, effect 3) still
	// carries base points 200; the hotfixed text is the source. The bonus
	// is +300%: damage x4 against a Frozen target. It applies against a
	// Frost Nova-frozen target and on a Fingers of Frost cast - see
	// isTargetFrozen.
	iceLanceFrozenMultiplier = 1 + iceLanceFrozenBonusPct/100
	// iceLanceFrozenBonusPct is the talent text's "300% increased damage".
	iceLanceFrozenBonusPct = 300.0

	// iceLanceSpellCoeff is every rank's spell power coefficient. The
	// client column is 0 and the generator's 0.4286 is only the
	// instant-spell convention. Forever beta logs (foreverlogs reports
	// 2762, 2763 and 2786, four Frost mages at 33-35) put it at 0.10 to
	// 0.12: each mage's spell power, read off their own Frostbolt (0.814),
	// Cone of Cold (0.129) and Frost Nova (0.029) hits, agrees within about
	// 10% across the three, and their unfrozen and frozen (/4) Ice Lance
	// hits sit 0.107 a point above the rank's base on average.
	iceLanceSpellCoeff = 0.107
)

func (mage *Mage) registerIceLanceSpell() {
	if !mage.Talents.IceLance {
		return
	}

	mage.IceLance = make([]*core.Spell, IceLanceRanks+1)
	cdTimer := mage.fireBlastCategory()

	// Rank 0 of the client's Ice Lance rows is not a player rank: its
	// base-damage column carries the spell id 400640 rather than a
	// damage figure, which is the signature of a trigger or NPC variant.
	// Player ranks start at 1.
	for rank := 1; rank <= IceLanceRanks; rank++ {
		config := mage.getIceLanceConfig(rank, cdTimer)

		if config.RequiredLevel <= int(mage.Level) {
			mage.IceLance[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

// iceLanceCooldown is every rank's cooldown: Fire Blast's 8 s category
// recovery on the timer they share. Client 1.60.1.70685 put rank 1
// (1312002) in Fire Blast's category 19 with an 8000 ms category recovery;
// the trained ranks (400640, 1240044-1240047) have no such row in the
// shipped tables, but Wowhead Forever's hotfix-applied tooltips show
// "8 sec cooldown" on every rank (the same data that raised the Frozen
// bonus to 300% over the shipped 200).
const iceLanceCooldown = 8 * time.Second

func (mage *Mage) getIceLanceConfig(rank int, cdTimer *core.Timer) core.SpellConfig {
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
			CD: core.Cooldown{Timer: cdTimer, Duration: iceLanceCooldown},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: iceLanceSpellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
			// "300% increased damage to Frozen targets" is the whole hit,
			// spell power share included, so it multiplies the spell's
			// damage for this cast. Shatter's part is crit chance, so the
			// two compose without an interaction rule.
			frozenMultiplier := 1.0
			if mage.isTargetFrozen(target) {
				frozenMultiplier = iceLanceFrozenMultiplier
			}
			// Winter's Chill counts the "Fingers of Frost empowered" Ice
			// Lances: the casts that spent a charge, not the ones a Frozen
			// aura made count. Read here, before the travel time clears it.
			empowered := mage.fingersFreezeCast
			spell.ApplyMultiplicativeDamageBonus(frozenMultiplier)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.ApplyMultiplicativeDamageBonus(1 / frozenMultiplier)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				if result.Landed() {
					spell.DealDamage(sim, result)
					if empowered {
						mage.procWintersChill(sim, target)
					}
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
