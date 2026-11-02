package warrior

import (
	"github.com/wowsims/classic/sim/core"
)

// executeRageConversionByRank is the damage each point of rage beyond the
// ability's own cost adds, by rank: 3, 6, 9, 12 and 15 (5308, 20658, 20660,
// 20661, 20662). The client's SpellEffect carries only the flat base
// (effect 3, ExecuteBaseDamage) and an effect 64 of 0; the per-rank figure
// is the client description's, as Wowhead Forever renders each rank's
// tooltip, and matches Era. Rank 0 is the level-1 stub.
var executeRageConversionByRank = [ExecuteRanks + 1]float64{0, 3, 6, 9, 12, 15}

// executeRageConversion is the damage per extra rage of an Execute rank.
func executeRageConversion(rank int) float64 {
	return executeRageConversionByRank[rank]
}

func (warrior *Warrior) registerExecuteSpell() {
	rank := rankAtLevel(ExecuteLevel[:], warrior.Level)
	flatDamage := ExecuteDamage[rank]
	ragePointDamage := executeRageConversion(rank)
	casterLevel := int(warrior.Level)
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
		// core.SpellFlagPassiveSpell used to sit here too. It tells
		// core/metrics_aggregator.go's addSpellMetrics to drop the
		// spell's OWN Casts count ("applied as a result of another
		// spell", not a direct player cast) - correct for a proc, wrong
		// for Execute, which is cast directly off the APL
		// (SpellFlagAPL, right below). ApplyEffects ran and dealt
		// damage every time the execute-phase window was open (checked
		// with a temporary debug counter), but the ladder's cast tally
		// reads the same Casts field addSpellMetrics was zeroing, so
		// every level reported Execute as never cast even though it
		// was landing hits and dealing damage the whole time.
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

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
				// Rank 0 (the level-1 stub a warrior below ExecuteLevel[1]=24
				// resolves to) is GCD-less in the client (gcd_ms 0); every
				// real rank takes the standard GCD.
				GCD: core.TernaryDuration(rank == 0, 0, core.GCDDefault),
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return sim.IsExecutePhase20()
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1.25,
		BonusCoefficient: 1,
		ClientBaseDamage: flatDamage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			extraRage := spell.Unit.CurrentRage()
			warrior.SpendRage(sim, extraRage, rageMetrics)
			// We must count this rage event if the spell itself cost 0,
			// otherwise we could end up with 0 events even though rage was spent.
			if spell.Cost.GetCurrentCost() > 0 {
				rageMetrics.Events--
			}

			baseDamage := flatDamage.Roll(sim, casterLevel) + ragePointDamage*extraRage

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
	rageMetrics = warrior.Execute.Cost.SpellCostFunctions.(*core.RageCost).ResourceMetrics
}
