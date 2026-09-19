package warrior

import (
	"github.com/wowsims/classic/sim/core"
)

// executeRageConversion is the damage each point of rage beyond the
// ability's own cost adds. The client's table carries the flat base
// (effect 3 of spell 20662, 600 at rank 5, which is ExecuteBaseDamage)
// but gives the conversion only as effect 64 with an amount of 0, so it
// stays typed at vanilla's 15 and unconfirmed until the validation job
// has something to compare against.
const executeRageConversion = 15.0

func (warrior *Warrior) registerExecuteSpell() {
	rank := rankAtLevel(ExecuteLevel[:], warrior.Level)
	flatDamage := ExecuteBaseDamage[rank][0]
	// The engine keeps spell 20662 rather than the generated
	// ExecuteSpellId[5]; they are the same id, and the rank-0 row the
	// client also carries (20647, a level-1 stub with an amount of 1) is
	// never the one a warrior casts.
	spellID := ExecuteSpellId[rank]

	var rageMetrics *core.ResourceMetrics
	warrior.Execute = warrior.RegisterSpell(BattleStance|BerserkerStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorExecute,
		ClassSpellMask: WarriorSpellMaskExecute,
		ActionID:       core.ActionID{SpellID: spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagPassiveSpell | SpellFlagOffensive,

		RequiredLevel: ExecuteLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			// Improved Execute's discount is a SpellMod in talents.go,
			// so this is the client's undiscounted cost.
			Cost:   rageCost(ExecuteManaCost[rank]),
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return sim.IsExecutePhase20()
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1.25,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			extraRage := spell.Unit.CurrentRage()
			warrior.SpendRage(sim, extraRage, rageMetrics)
			// We must count this rage event if the spell itself cost 0,
			// otherwise we could end up with 0 events even though rage was spent.
			if spell.Cost.GetCurrentCost() > 0 {
				rageMetrics.Events--
			}

			baseDamage := flatDamage + executeRageConversion*(extraRage)

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
	rageMetrics = warrior.Execute.Cost.SpellCostFunctions.(*core.RageCost).ResourceMetrics
}
