package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// ambushLearnLevels are Ambush's six rank learn levels; source:
// 1.60.1.70009 client spell data ("Ambush", ranks 1-6). Each rank's level
// also carries a duplicate id in the 460000s (e.g. 462718 beside rank 1's
// 8676); those are a clone of the same rank, not an extra rank -- dropped.
var ambushLearnLevels = []int{18, 26, 34, 42, 50, 58}

// ambushSpellID is Ambush's rank -> spell id, index 0 unused.
var ambushSpellID = [7]int32{0, 8676, 8724, 8725, 11267, 11268, 11269}

func (rogue *Rogue) registerAmbushSpell() {
	rank := core.HighestRankAtLevel(ambushLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	flatDamage := AmbushDamage[rank]
	casterLevel := int(rogue.Level)
	spellID := ambushSpellID[rank]

	damageMultiplier := 2.5 * rogue.opportunity()

	rogue.Ambush = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_RogueAmbush,
		ClassSpellMask: RogueSpellMaskAmbush,
		ActionID:       core.ActionID{SpellID: spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          rogue.builderFlags() | SpellFlagColdBlooded,
		RequiredLevel:  ambushLearnLevels[rank-1],

		EnergyCost: core.EnergyCostOptions{
			Cost:   60,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			if !rogue.HasDagger(core.MainHand) {
				return false
			}
			return rogue.canOpen()
		},

		BonusCritRating:  15 * core.CritRatingPerCritChance * float64(rogue.Talents.ImprovedAmbush),
		DamageMultiplier: damageMultiplier,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,
		ClientBaseDamage: flatDamage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.spendOpener(sim)
			baseDamage := flatDamage.Roll(sim, casterLevel) + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialNoBlockDodgeParry)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
