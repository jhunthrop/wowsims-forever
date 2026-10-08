package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// eviscerateLearnLevels are Eviscerate's nine rank learn levels; source:
// 1.60.1.70009 client spell data ("Eviscerate", ranks 1-9; the level-1
// rank-0 ids are internal copies, not player ranks).
var eviscerateLearnLevels = []int{1, 8, 16, 24, 32, 40, 48, 56, 60}

// eviscerateSpellID is Eviscerate's rank -> spell id, index 0 unused.
// Rank 9 only exists with AQ content, same as the old code's ternary.
var eviscerateSpellID = [10]int32{0, 2098, 6760, 6761, 6762, 8623, 8624, 11299, 11300, core.TernaryInt32(core.IncludeAQ, 31016, 11300)}

// eviscerateComboDamageBonus is Eviscerate's rank -> damage per combo point,
// index 0 unused: the client's EffectPointsPerResource (1.60.1.70009
// SpellEffect.csv, ids 2098-31016), which the vendored spellconst does not
// carry. The roll it adds to is EviscerateDamage. The final slot is rank 9
// with AQ content, otherwise rank 8's term (the id the slot casts).
var eviscerateComboDamageBonus = [10]float64{0, 5, 11, 19, 31, 45, 71, 110, 151, core.TernaryFloat64(core.IncludeAQ, 170, 151)}

func (rogue *Rogue) registerEviscerate() {
	rank := core.HighestRankAtLevel(eviscerateLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	damage := EviscerateDamage[rank]
	casterLevel := int(rogue.Level)
	comboDamageBonus := eviscerateComboDamageBonus[rank]
	spellID := eviscerateSpellID[rank]

	rogue.Eviscerate = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueEviscerate,
		ActionID:      core.ActionID{SpellID: spellID},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         rogue.finisherFlags() | SpellFlagColdBlooded,
		MetricSplits:  6,
		RequiredLevel: eviscerateLearnLevels[rank-1],

		EnergyCost: core.EnergyCostOptions{
			Cost:   35,
			Refund: 0,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(spell.Unit.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		DamageMultiplier: 1 +
			(improvedEviscerateMultiplier[rankIndex(rogue.Talents.ImprovedEviscerate, improvedEviscerateMultiplier[:])] - 1) +
			rogue.aggressionBonus(),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			comboPoints := rogue.ComboPoints()
			baseDamage := damage.Roll(sim, casterLevel) + comboDamageBonus*float64(comboPoints) +
				0.03*float64(comboPoints)*spell.MeleeAttackPower(target)

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				rogue.SpendComboPoints(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}

			spell.DealDamage(sim, result)
		},
	})
	rogue.Finishers = append(rogue.Finishers, rogue.Eviscerate)
}
