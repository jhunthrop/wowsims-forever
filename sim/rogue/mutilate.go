package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Mutilate is a Forever-only Assassination talent (tier 4, single rank,
// no learnable ranks of its own): talent node 105709, spell 1310707.
// Gated purely on the talent point, the same way registerGhostlyStrikeSpell
// gates on rogue.Talents.GhostlyStrike, since the client carries no
// separate level-learned rank table for it the way Sinister Strike or
// Ambush have.
//
// The client's own data models the strike as the outer spell (1310707,
// a "Trigger Spell" dummy effect pointing at two identical sub-spells,
// 1310705 and 1310706 -- one per hand) rather than one spell dealing
// damage twice, so this file keeps that shape: MutilateMH/MutilateOH are
// the two hit spells (their own ProcMask so each hand's landed hit can
// independently proc that weapon's poison, matching Deadly/Instant/Wound
// Poison's OnSpellHitDealt hooks, and their own SpellMetrics so each
// hand's damage/hit rate is inspectable on its own), and the talented
// button only spends the energy and awards the combo points.
//
// mutilateFlatDamageBonus: the client's per-hand effect (1310705/1310706
// effect index 0, effect 121, amount 23) is a real tuned number, not a
// dummy/server-side-script placeholder, so per this lane's source-of-truth
// rule it wins over the talent tooltip's own stated "17.25" -- the
// tooltip text is kept here only as a comment for a future reader who
// diffs against Wowhead. mutilateWeaponDamagePct (75%) matches both the
// tooltip and the sub-spells' effect index 1 (effect 31, amount 75).
const (
	mutilateEnergyCost      = 60.0
	mutilateFlatDamageBonus = 23.0 // client spellconst; talent tooltip states 17.25 for the same hit.
	mutilateWeaponDamagePct = 0.75
	// mutilatePoisonedTargetMultiplier: "Damage increased by 20% against
	// Poisoned targets" (talent tooltip, spell 1310707). No client effect
	// carries this as a number, so the tooltip is the only source.
	mutilatePoisonedTargetMultiplier = 1.20
	mutilateComboPointsAwarded       = 2
)

func (rogue *Rogue) registerMutilateSpell() {
	if !rogue.Talents.Mutilate {
		return
	}

	// Opportunity (talent node 105760): "Increases the damage dealt by
	// your Backstab, Garrote, Ambush, and Mutilate abilities by 5%/10%."
	// Two ranks per the client's tree, unlike the stale 4/8/12/16/20%,
	// five-rank array ambush.go reuses for the same talent -- that array
	// predates the Forever rewrite and is out of scope for this lane, so
	// it is not copied here.
	opportunityMultiplier := []float64{1, 1.05, 1.10}[rogue.Talents.Opportunity]

	rogue.MutilateMH = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 1310705},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell,

		CritDamageBonus:  rogue.lethality(),
		DamageMultiplier: opportunityMultiplier,
		ThreatMultiplier: 1,
	})

	rogue.MutilateOH = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 1310706},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeOHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell,

		CritDamageBonus:  rogue.lethality(),
		DamageMultiplier: opportunityMultiplier,
		ThreatMultiplier: 1,
	})

	rogue.Mutilate = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_RogueMutilate,
		ActionID:    core.ActionID{SpellID: 1310707},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeOHSpecial,
		Flags:       rogue.builderFlags(),

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
			// "Instantly attacks with both weapons": needs a weapon in
			// each hand, not daggers specifically -- the tooltip names no
			// weapon-type restriction the way Ambush/Backstab do.
			return rogue.AutoAttacks.IsDualWielding
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
			mhDamage := poisonedMultiplier * (mutilateFlatDamageBonus + mutilateWeaponDamagePct*rogue.AutoAttacks.MH().CalculateNormalizedWeaponDamage(sim, ap))
			ohDamage := poisonedMultiplier * (mutilateFlatDamageBonus + mutilateWeaponDamagePct*rogue.AutoAttacks.OH().CalculateNormalizedWeaponDamage(sim, ap))

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
