package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Primal Bite (talent node 104949; spells 407995, 1238069, 1238070 and
// 1238073, learned at 25, 36, 48 and 60; build 1.60.1.70009): "Bite the
// target, dealing 100% normal damage plus 26 and generating a high amount
// of threat." The rank text's 26 is rank 1's flat amount (38, 59 and 77
// after it). Every rank costs 20 rage (cost 200, less Ferocity's 1 a rank),
// is on the global cooldown and has a 6 second cooldown of its own. The
// client states a weapon-damage effect (effect 58, the flat amount) and a
// 100% weapon-percent effect (effect 31), so the hit is the paw's swing
// plus the flat amount, which "normal damage" is read as.
//
// Berserk (node 104956): "Causes your Primal Bite ability to strike up to
// 3 targets, removes its cooldown ..." (primalBiteBerserkTargets).
//
// unconfirmed: the threat multiplier (HighThreatMultiplier).
const (
	primalBiteRageCost = 20.0
	// primalBiteCooldown is the client's category cooldown on every rank.
	primalBiteCooldown   = 6 * time.Second
	primalBiteMissRefund = 0.8
	// primalBiteBerserkTargets is how many enemies Primal Bite strikes
	// under Berserk.
	primalBiteBerserkTargets = 3
)

// PrimalBiteFlatDamage is the weapon-damage effect's flat amount per rank.
var PrimalBiteFlatDamage = clientdamage.FromTable(PrimalBiteBaseDamage[:], PrimalBitePointsPerLevel[:], PrimalBiteLevel[:], PrimalBiteMaxLevel[:])

func (druid *Druid) registerPrimalBiteSpell() {
	if !druid.Talents.PrimalBite {
		return
	}
	rank := core.HighestRankAtLevel(PrimalBiteLevel[1:], druid.Level)
	if rank == 0 {
		return
	}
	flatDamage := PrimalBiteFlatDamage[rank]
	casterLevel := int(druid.Level)

	// Pool-sized ceiling, live-bounded loop; see registerSwipeBearSpell.
	results := make([]*core.SpellResult, min(primalBiteBerserkTargets, len(druid.Env.Encounter.AllTargetUnits)))

	druid.PrimalBite = druid.RegisterSpell(Bear, core.SpellConfig{
		SpellCode:      SpellCode_DruidPrimalBite,
		ClassSpellMask: DruidSpellMaskPrimalBite,
		ActionID:       core.ActionID{SpellID: PrimalBiteSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          SpellFlagOmen | core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		Rank:          rank,
		RequiredLevel: PrimalBiteLevel[rank],

		RageCost: core.RageCostOptions{
			Cost:   primalBiteRageCost - ferocityRageDiscount(druid.Talents.Ferocity),
			Refund: primalBiteMissRefund,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: primalBiteCooldown,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: HighThreatMultiplier,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			numHits := 1
			if druid.BerserkAura != nil && druid.BerserkAura.IsActive() {
				numHits = min(len(results), len(sim.Encounter.TargetUnits))
			}

			for idx := 0; idx < numHits; idx++ {
				baseDamage := flatDamage.Roll(sim, casterLevel) +
					spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
				baseDamage *= druid.RendAndTearMultiplier(target)
				results[idx] = spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}

			if !results[0].Landed() {
				spell.IssueRefund(sim)
			}
			for _, result := range results[:numHits] {
				spell.DealDamage(sim, result)
			}
		},
	})
}
