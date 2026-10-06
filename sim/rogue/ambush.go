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

// ambushFlatDamageBonus is Ambush's rank -> flat damage bonus, index 0
// unused; source: 1.60.1.70009 spellconst (each rank's own effect 121
// amount: 28, 40, 50, 74, 92, 116). All six ranks have a real, per-rank
// client number.
var ambushFlatDamageBonus = [7]float64{0, 28, 40, 50, 74, 92, 116}

func (rogue *Rogue) registerAmbushSpell() {
	rank := core.HighestRankAtLevel(ambushLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	flatDamageBonus := ambushFlatDamageBonus[rank]
	spellID := ambushSpellID[rank]

	damageMultiplier := 2.5 * []float64{1, 1.04, 1.08, 1.12, 1.16, 1.2}[rogue.Talents.Opportunity]

	rogue.Ambush = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_RogueAmbush,
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       rogue.builderFlags(),

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
			if rogue.IsStealthed() {
				return true
			}
			// Cutthroat (talents.go's applyCutthroat): a landed Backstab
			// has a chance to let the next Ambush within 10 sec skip the
			// stealth requirement. CutthroatAura is nil for a build with
			// no points in the talent, so this is a no-op there.
			return rogue.CutthroatAura != nil && rogue.CutthroatAura.IsActive()
		},

		BonusCritRating:  15 * core.CritRatingPerCritChance * float64(rogue.Talents.ImprovedAmbush),
		DamageMultiplier: damageMultiplier,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Consume the Cutthroat bypass before BreakStealth, which
			// would otherwise make "not stealthed" ambiguous between
			// "never stealthed" and "was stealthed, now broken".
			if !rogue.IsStealthed() && rogue.CutthroatAura != nil && rogue.CutthroatAura.IsActive() {
				rogue.CutthroatAura.Deactivate(sim)
			}
			rogue.BreakStealth(sim)
			baseDamage := (flatDamageBonus + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target)))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialNoBlockDodgeParry)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
