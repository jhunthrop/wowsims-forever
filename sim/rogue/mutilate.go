package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Mutilate is granted by a single Assassination talent point (node
// 105709, tier 6 in 70755) but is NOT a single, unranked ability once
// learned: the client carries three player ranks, learned by character
// level the same way Sinister Strike or Ambush are (SkillLineAbility
// 49379/54140/54141: 399956@40, 1241582@50, 1241584@60). 70755 deleted the
// old level-30 rank 1310707 and its hand spells 1310705/1310706 (its
// SkillLineAbility row is AcquireMethod 3, not learnable), so the chain
// starts at 40.
//
// The client's own data models each rank's strike as the outer spell
// (a "Trigger Spell" dummy effect pointing at two identical sub-spells,
// one per hand) rather than one spell dealing damage twice, so this
// file keeps that shape per rank: mutilateMHSpellID/mutilateOHSpellID
// are the two hit spells (their own ProcMask so each hand's landed hit
// can independently proc that weapon's poison, matching
// Deadly/Instant/Wound Poison's OnSpellHitDealt hooks, and their own
// SpellMetrics so each hand's damage/hit rate is inspectable on its
// own), and the talented button only spends the energy and awards the
// combo points.
//
// Each hand spell is e0 effect 121 (normalized weapon damage plus a flat
// 44/63/88 at ranks 1-3, MutilateDamage) under e1 effect 31 at 90%: the
// talent text's "plus an additional 40" is rank 1's 44 x 0.90.
const mutilateRanks = 3

var mutilateSpellID = [mutilateRanks + 1]int32{0, 399956, 1241582, 1241584}
var mutilateMHSpellID = [mutilateRanks + 1]int32{0, 399960, 1241585, 1241586}
var mutilateOHSpellID = [mutilateRanks + 1]int32{0, 399961, 1241588, 1241590}

// mutilateLearnLevels is core.HighestRankAtLevel's input shape: rank r
// (1-based) is learned at mutilateLearnLevels[r-1], no rank-0 placeholder
// (unlike the by-rank arrays above), the same convention
// eviscerate.go's eviscerateLearnLevels and ambush.go's
// ambushLearnLevels already use.
var mutilateLearnLevels = []int{40, 50, 60}

const (
	mutilateEnergyCost      = 60.0
	mutilateWeaponDamagePct = 0.90
	// mutilatePoisonedTargetMultiplier: "Damage increased by 20% against
	// Poisoned targets" (399956 e3, effect 3 bp 20, a server script).
	mutilatePoisonedTargetMultiplier = 1.20
	mutilateComboPointsAwarded       = 2
)

func (rogue *Rogue) registerMutilateSpell() {
	if !rogue.Talents.Mutilate {
		return
	}

	rank := core.HighestRankAtLevel(mutilateLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}
	flatDamage := MutilateDamage[rank]
	casterLevel := int(rogue.Level)

	// Opportunity (talent node 105760): the same table (client_values.go)
	// serves Backstab, Garrote and Ambush.
	opportunityDamage := rogue.opportunity()

	rogue.MutilateMH = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: mutilateMHSpellID[rank]},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		// Cold Blood's crit bonus rides on the hand spells that roll the
		// hits; the main hand leaves it for the off hand to spend.
		// They carry the builder flag so a crit procs Seal Fate; its
		// 500 ms cooldown keeps that to one point per Mutilate.
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell | SpellFlagColdBlooded | SpellFlagBuilder,

		CritDamageBonus:  rogue.lethality(),
		DamageMultiplier: opportunityDamage,
		ThreatMultiplier: 1,
	})

	rogue.MutilateOH = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: mutilateOHSpellID[rank]},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeOHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell | SpellFlagColdBlooded | SpellFlagBuilder,

		CritDamageBonus:  rogue.lethality(),
		DamageMultiplier: opportunityDamage,
		ThreatMultiplier: 1,
	})

	rogue.Mutilate = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_RogueMutilate,
		ClassSpellMask: RogueSpellMaskMutilate,
		ActionID:       core.ActionID{SpellID: mutilateSpellID[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeOHSpecial,
		Flags:          rogue.builderFlags() | SpellFlagColdBlooded,

		RequiredLevel:    mutilateLearnLevels[rank-1],
		Rank:             rank,
		ClientBaseDamage: flatDamage.Range(casterLevel),

		EnergyCost: core.EnergyCostOptions{
			Cost:   mutilateEnergyCost,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			// A dagger in each hand: every Mutilate rank's
			// SpellEquippedItems row is item class 2, subclass mask 32768
			// (the dagger bit), the same row Backstab carries.
			return rogue.AutoAttacks.IsDualWielding && rogue.HasDagger(core.MainHand) && rogue.HasDagger(core.OffHand)
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			poisonedMultiplier := 1.0
			if rogue.TargetHasRoguePoison(target) {
				poisonedMultiplier = mutilatePoisonedTargetMultiplier
			}

			// Both hands read their own weapon's normalized damage
			// directly off AutoAttacks.MH()/OH(), not the unit-level
			// MHNormalizedWeaponDamage/OHNormalizedWeaponDamage helpers:
			// the latter bakes in the auto-attack-only 50% off-hand
			// penalty (core/attack.go), which does not apply to a
			// special ability that explicitly swings the off-hand.
			ap := spell.MeleeAttackPower(target)
			mhDamage := poisonedMultiplier * mutilateHandDamage(flatDamage.Roll(sim, casterLevel), rogue.AutoAttacks.MH().CalculateNormalizedWeaponDamage(sim, ap))
			ohDamage := poisonedMultiplier * mutilateHandDamage(flatDamage.Roll(sim, casterLevel), rogue.mutilateOHWeaponDamage(sim, ap))

			mhResult := rogue.MutilateMH.CalcAndDealDamage(sim, target, mhDamage, rogue.MutilateMH.OutcomeMeleeWeaponSpecialHitAndCrit)
			ohResult := rogue.MutilateOH.CalcAndDealDamage(sim, target, ohDamage, rogue.MutilateOH.OutcomeMeleeWeaponSpecialHitAndCrit)

			if mhResult.Landed() || ohResult.Landed() {
				rogue.AddComboPoints(sim, mutilateComboPointsAwarded, target, spell.ComboPointMetrics())
			}
			if !mhResult.Landed() && !ohResult.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}

// mutilateHandDamage is one hand's Mutilate damage before multipliers: the
// flat bonus sits inside the 90% weapon damage (client effect 121, weapon
// plus 44, under effect 31 at 90%; the tooltip's "plus 40" is 44 x 0.90),
// the way Backstab's flat sits inside its 150%.
func mutilateHandDamage(flat, normalizedWeaponDamage float64) float64 {
	return mutilateWeaponDamagePct * (flat + normalizedWeaponDamage)
}

// TargetHasRoguePoison reports whether the caster's own poisons are
// currently applied to target, for Mutilate's "against Poisoned targets"
// damage bonus. Deadly Poison (a stacking dot) and Wound Poison (a
// stacking debuff) both leave a checkable aura on the target; Instant
// Poison is a single hit with nothing left behind to check, so it cannot
// contribute here.
func (rogue *Rogue) TargetHasRoguePoison(target *core.Unit) bool {
	if rogue.deadlyPoisonTick != nil && rogue.deadlyPoisonTick.Dot(target).IsActive() {
		return true
	}
	return rogue.woundPoisonDebuffAuras.Get(target).IsActive()
}

// mutilateOHWeaponDamage is the off-hand strike's normalized weapon damage,
// raised by Dual Wield Specialization: 13715 is aura 122, the off-hand damage
// percent, which scales the off-hand weapon's damage for every off-hand
// strike, not only the white swing.
func (rogue *Rogue) mutilateOHWeaponDamage(sim *core.Simulation, attackPower float64) float64 {
	return rogue.AutoAttacks.OH().CalculateNormalizedWeaponDamage(sim, attackPower) * rogue.dwsMultiplier()
}
