package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Mutilate is granted by a single Assassination talent point (node
// 105709) but is NOT a single, unranked ability once learned: the
// client carries four player ranks, learned by character level the
// same way Sinister Strike or Ambush are (spellranks.json's "Mutilate"
// chain: 1310707@30, 399956@40, 1241582@50, 1241584@60). A duplicate
// rank-1 row, 1329@40, shares rank 1 with 1310707 and triggers a
// different, older sub-spell pair (5374) the rest of this chain does
// not use, the same kind of dedup artifact heroic_strike_cleave.go's
// rank 3 has, so it is skipped. Before this, the whole chain outside
// rank 1 was missing: the rotation's castSpell (rewritten to the rank
// the character has actually learned, same as every other ranked
// ability) could never find a level-40+ Mutilate, so the button never
// fired past level 30 and, since it is Assassination's combo-point
// generator, Eviscerate and Slice and Dice starved with it.
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
// mutilateFlatDamageBonus is each rank's per-hand effect (effect 121,
// amount 23/33/48/67 at ranks 1-4) - a real tuned number, not a
// dummy/server-side-script placeholder, so per this lane's
// source-of-truth rule it wins over the talent tooltip's own stated
// "17.25" for rank 1 - kept here only as a comment for a future reader
// who diffs against Wowhead. mutilateWeaponDamagePct (75%) matches both
// the tooltip and every rank's sub-spells' effect index 1 (effect 31,
// amount 75) and does not vary by rank.
const mutilateRanks = 4

var mutilateSpellID = [mutilateRanks + 1]int32{0, 1310707, 399956, 1241582, 1241584}
var mutilateMHSpellID = [mutilateRanks + 1]int32{0, 1310705, 399960, 1241585, 1241586}
var mutilateOHSpellID = [mutilateRanks + 1]int32{0, 1310706, 399961, 1241588, 1241590}
var mutilateFlatDamageBonus = [mutilateRanks + 1]float64{0, 23, 33, 48, 67}

// mutilateLearnLevels is core.HighestRankAtLevel's input shape: rank r
// (1-based) is learned at mutilateLearnLevels[r-1], no rank-0 placeholder
// (unlike the by-rank arrays above), the same convention
// eviscerate.go's eviscerateLearnLevels and ambush.go's
// ambushLearnLevels already use.
var mutilateLearnLevels = []int{30, 40, 50, 60}

const (
	mutilateEnergyCost      = 60.0
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

	rank := core.HighestRankAtLevel(mutilateLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}
	flatDamageBonus := mutilateFlatDamageBonus[rank]

	// Opportunity (talent node 105760): "Increases the damage dealt by
	// your Backstab, Garrote, Ambush, and Mutilate abilities by 5%/10%."
	// Two ranks per the client's tree, unlike the stale 4/8/12/16/20%,
	// five-rank array ambush.go reuses for the same talent -- that array
	// predates the Forever rewrite and is out of scope for this lane, so
	// it is not copied here.
	opportunityMultiplier := []float64{1, 1.05, 1.10}[rogue.Talents.Opportunity]

	rogue.MutilateMH = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: mutilateMHSpellID[rank]},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell,

		CritDamageBonus:  rogue.lethality(),
		DamageMultiplier: opportunityMultiplier,
		ThreatMultiplier: 1,
	})

	rogue.MutilateOH = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: mutilateOHSpellID[rank]},
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
		ActionID:    core.ActionID{SpellID: mutilateSpellID[rank]},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeOHSpecial,
		Flags:       rogue.builderFlags() | SpellFlagColdBlooded,

		RequiredLevel: mutilateLearnLevels[rank-1],
		Rank:          rank,

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
			mhDamage := poisonedMultiplier * (flatDamageBonus + mutilateWeaponDamagePct*rogue.AutoAttacks.MH().CalculateNormalizedWeaponDamage(sim, ap))
			ohDamage := poisonedMultiplier * (flatDamageBonus + mutilateWeaponDamagePct*rogue.AutoAttacks.OH().CalculateNormalizedWeaponDamage(sim, ap))

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
