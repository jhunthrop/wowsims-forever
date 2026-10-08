package druid

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// clientRageCostScale: the client stores a rage cost in tenths of a point
// (Maul's 150 is 15 rage), and mana and energy costs in whole points.
const clientRageCostScale = 10

// clientRageCost converts a spellconst rage cost to rage points.
func clientRageCost(clientCost float64) float64 {
	return clientCost / clientRageCostScale
}

// MaulFlatDamage is Maul's weapon-damage effect (effect 58): "Maul: the
// next melee attack deals weapon damage plus N", N being the effect's amount
// (18 at rank 1, 128 at rank 7), which the generated table carries per rank.
var MaulFlatDamage = clientdamage.FromTable(MaulBaseDamage[:], MaulPointsPerLevel[:], MaulLevel[:], MaulMaxLevel[:])

// Maul is a Bear Form ability that replaces the next main-hand swing: the
// APL "casts" MaulQueueSpell to queue it (the rage must be on hand), and the
// swing that follows is the Maul if the rage is still there, or an ordinary
// swing if not. Costs 15 rage (cost 150, every rank) less Ferocity's 1 a
// rank; a miss refunds 80% of it.
const maulMissRefund = 0.8

func (druid *Druid) registerMaulSpell() {
	rank := core.HighestRankAtLevel(MaulLevel[1:], druid.Level)
	if rank == 0 {
		return
	}
	flatDamage := MaulFlatDamage[rank]
	casterLevel := int(druid.Level)

	druid.Maul = druid.RegisterSpell(Bear, core.SpellConfig{
		SpellCode:      SpellCode_DruidMaul,
		ClassSpellMask: DruidSpellMaskMaul,
		ActionID:       core.ActionID{SpellID: MaulSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeMHAuto,
		Flags:          SpellFlagOmen | core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete,

		Rank:          rank,
		RequiredLevel: MaulLevel[rank],

		RageCost: core.RageCostOptions{
			Cost:   clientRageCost(MaulManaCost[rank]) - ferocityRageDiscount(druid.Talents.Ferocity),
			Refund: maulMissRefund,
		},

		DamageMultiplierAdditive: druid.savageFuryDamageMultiplier(),
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
		BonusCoefficient:         1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := flatDamage.Roll(sim, casterLevel) +
				spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			baseDamage *= druid.RendAndTearMultiplier(target)

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			if !result.Landed() {
				spell.IssueRefund(sim)
			}

			druid.MaulQueueAura.Deactivate(sim)
		},
	})

	druid.MaulQueueAura = druid.RegisterAura(core.Aura{
		Label:    "Maul Queue Aura",
		Duration: core.NeverExpires,
	})

	druid.MaulQueueSpell = druid.RegisterSpell(Bear, core.SpellConfig{
		ActionID:    druid.Maul.WithTag(1),
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		Rank: rank,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return !druid.MaulQueueAura.IsActive() &&
				druid.CurrentRage() >= druid.Maul.Cost.GetCurrentCost() &&
				!druid.IsCasting(sim)
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.MaulQueueAura.Activate(sim)
		},
	})

	druid.ReplaceBearMHFunc = druid.maulReplaceMH
}

// maulReplaceMH swaps a queued Maul in for the next main-hand swing.
func (druid *Druid) maulReplaceMH(sim *core.Simulation, mhSwingSpell *core.Spell) *core.Spell {
	if !druid.MaulQueueAura.IsActive() {
		return mhSwingSpell
	}

	if !druid.Maul.Spell.CanCast(sim, druid.CurrentTarget) {
		druid.MaulQueueAura.Deactivate(sim)
		return mhSwingSpell
	}

	return druid.Maul.Spell
}
