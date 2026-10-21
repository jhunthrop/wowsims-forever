package warrior

import (
	"github.com/wowsims/classic/sim/core"
)

// sunderArmorThreat is each rank's flat threat, the client's effect 63
// (SPELL_EFFECT_THREAT) amount on spells 7386, 7405, 8380, 11596 and
// 11597: 34, 75, 117, 158 and 206. The formula the fork used before
// (2.25 x 2 x the spell's level) gave 261 at rank 5, which is vanilla's
// figure and not Forever's. Defensive Stance and Defiance multiply it at
// run time like every other threat.
var sunderArmorThreat = [SunderArmorRanks + 1]float64{0, 34, 75, 117, 158, 206}

func (warrior *Warrior) registerSunderArmorSpell() {
	warrior.SunderArmorAuras = warrior.NewEnemyAuraArray(core.SunderArmorAura)

	rank := rankAtLevel(SunderArmorLevel[:], warrior.Level)
	spellID := SunderArmorSpellId[rank]

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
		FlatThreatBonus:  sunderArmorThreat[rank],

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
