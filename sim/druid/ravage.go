package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// ravageLearnLevels are Ravage's four rank learn levels; source:
// 1.60.1.70009 client spell data ("Ravage", ranks 1-4).
var ravageLearnLevels = []int{32, 42, 50, 58}

// ravageSpellID is Ravage's rank -> spell id, index 0 unused.
var ravageSpellID = [5]int32{0, 6785, 6787, 9866, 9867}

// ravageFlatDamageBonus is Ravage's rank -> flat damage bonus (effect 0
// base points), index 0 unused. Source: 1.60.1.70009 spellconst/druid.json.
var ravageFlatDamageBonus = [5]float64{0, 42, 62, 78, 98}

// ravageDamageMultiplier is Ravage's weapon-damage multiplier on every
// rank (effect 1, amount 350 -> 3.5x); source: 1.60.1.70009 spellconst.
const ravageDamageMultiplier = 3.5

func (druid *Druid) registerRavageSpell() {
	rank := core.HighestRankAtLevel(ravageLearnLevels, druid.Level)
	if rank == 0 {
		return
	}

	flatDamageBonus := ravageFlatDamageBonus[rank]

	druid.Ravage = druid.RegisterSpell(Cat, core.SpellConfig{
		SpellCode:      SpellCode_DruidRavage,
		ClassSpellMask: DruidSpellMaskRavage,
		ActionID:       core.ActionID{SpellID: ravageSpellID[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOmen | SpellFlagBuilder,

		RequiredLevel: ravageLearnLevels[rank-1],

		EnergyCost: core.EnergyCostOptions{
			// Every client rank of Ravage costs 60 energy; source:
			// 1.60.1.70009 spellconst/druid.json.
			Cost:   60,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},
		// Ravage requires Prowl: the client data carries no such flag (it's
		// a scripted requirement, the same way Ambush's stealth requirement
		// in sim/rogue/ambush.go isn't in its own spellconst either), so
		// this mirrors the rogue precedent directly.
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return druid.IsProwling()
		},

		DamageMultiplier: ravageDamageMultiplier,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

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
