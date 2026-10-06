package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// clawLearnLevels are Claw's five rank learn levels; source: 1.60.1.70009
// client spell data ("Claw", ranks 1-5, spellranks.json).
var clawLearnLevels = []int{20, 28, 38, 48, 58}

// clawSpellID is Claw's rank -> spell id, index 0 unused.
var clawSpellID = [6]int32{0, 1082, 3029, 5201, 9849, 9850}

// clawFlatDamageBonus is Claw's rank -> flat damage bonus (effect 0 base
// points), index 0 unused. Source: 1.60.1.70009 client spellconst/druid.json.
var clawFlatDamageBonus = [6]float64{0, 27, 39, 57, 88, 115}

func (druid *Druid) registerClawSpell() {
	rank := core.HighestRankAtLevel(clawLearnLevels, druid.Level)
	if rank == 0 {
		return
	}

	flatDamageBonus := clawFlatDamageBonus[rank]

	druid.Claw = druid.RegisterSpell(Cat, core.SpellConfig{
		SpellCode:      SpellCode_DruidClaw,
		ClassSpellMask: DruidSpellMaskClaw,
		ActionID:       core.ActionID{SpellID: clawSpellID[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOmen | SpellFlagBuilder,

		Rank:          rank,
		RequiredLevel: clawLearnLevels[rank-1],

		EnergyCost: core.EnergyCostOptions{
			// Every client rank of Claw costs 45 energy; source: 1.60.1.70009
			// spellconst/druid.json.
			Cost:   45 - 1*float64(druid.Talents.Ferocity),
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},

		DamageMultiplierAdditive: 1 + 0.1*float64(druid.Talents.SavageFury),
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
		BonusCoefficient:         1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.BreakProwl(sim)

			baseDamage := flatDamageBonus + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				druid.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
