package warrior

import (
	"github.com/wowsims/classic/sim/core"
)

func (warrior *Warrior) registerSunderArmorSpell() {
	warrior.SunderArmorAuras = warrior.NewEnemyAuraArray(core.SunderArmorAura)

	rank := rankAtLevel(SunderArmorLevel[:], warrior.Level)
	spellID := SunderArmorSpellId[rank]

	spell_level := SunderArmorLevel[rank]

	var canApplySunder bool

	warrior.SunderArmor = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: spellID},
		ClassSpellMask: WarriorSpellMaskSunderArmor,
		SpellSchool:    core.SpellSchoolPhysical,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: SunderArmorLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			// Improved Sunder Armor's discount is a SpellMod in
			// talents.go, so this is the client's undiscounted cost.
			// SunderArmorBaseDamage is the -450 armour the aura strips,
			// not damage; the ability itself deals none.
			Cost:   rageCost(SunderArmorManaCost[rank]),
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			sa := warrior.SunderArmorAuras.Get(target)
			if sa.IsActive() {
				canApplySunder = true
			} else if sa.ExclusiveEffects[0].Category.AnyActive() {
				canApplySunder = false
			} else {
				canApplySunder = true
			}
			return canApplySunder
		},

		ThreatMultiplier: 1,
		FlatThreatBonus:  2.25 * 2 * float64(spell_level),

		RelatedAuras: []core.AuraArray{warrior.SunderArmorAuras},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMeleeWeaponSpecialNoCrit) // Cannot be blocked
			if !result.Landed() {
				spell.IssueRefund(sim)
				return
			}

			if canApplySunder {
				sa := warrior.SunderArmorAuras.Get(target)
				sa.Activate(sim)
				sa.AddStack(sim)
			}
		},
	})
}
