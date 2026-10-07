package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Revenge is the one warrior ability constants_auto_gen.go does not
// cover: the generator skips a spell whose <Name>Ranks it would collide
// with, and this hand-written rank list is that collision (the generated
// file records it as `skipped: "Revenge" already has a hand-written
// RevengeRanks elsewhere in this package`). Its damage is not the
// vanilla table any more: RevengeDamage (client_damage.go) carries the
// client's own roll, which spellconst_damage_test.go checks. Freeing
// the name so the generator can emit Revenge belongs to the
// warrior-protection spec's task together with the rest of that tree.
const RevengeRanks = 6

var RevengeSpellId = [RevengeRanks + 1]int32{0, 6572, 6574, 7379, 11600, 11601, 25288}
var RevengeLevel = [RevengeRanks + 1]int{0, 14, 24, 34, 44, 54, 60}

func (warrior *Warrior) registerRevengeSpell(cdTimer *core.Timer) {
	rank := core.TernaryInt(core.IncludeAQ, RevengeRanks, RevengeRanks-1)
	actionID := core.ActionID{SpellID: RevengeSpellId[rank]}
	has2pcDreadnaught := warrior.HasSetBonus(ItemSetDreadnaughtsBattlegear, 2)
	damage := RevengeDamage[rank]
	casterLevel := int(warrior.Level)
	dreadnaughtBonus := core.TernaryFloat64(has2pcDreadnaught, 75, 0)
	revengeLevel := float64(RevengeLevel[rank])

	warrior.revengeProcAura = warrior.RegisterAura(core.Aura{
		Label:    "Revenge",
		Duration: 5 * time.Second,
		ActionID: actionID,
	})

	warrior.RegisterAura(core.Aura{
		Label:    "Revenge Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.Outcome.Matches(core.OutcomeBlock | core.OutcomeDodge | core.OutcomeParry) {
				warrior.revengeProcAura.Activate(sim)
			}
		},
	})

	warrior.Revenge = warrior.RegisterSpell(DefensiveStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorRevenge,
		ClassSpellMask: WarriorSpellMaskRevenge,
		ActionID:       actionID,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: int(revengeLevel),

		RageCost: core.RageCostOptions{
			Cost:   5,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: time.Second * 5,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.revengeProcAura.IsActive()
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 2.25,
		FlatThreatBonus:  2.25 * 2 * revengeLevel,
		BonusCoefficient: 1,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := damage.Roll(sim, casterLevel) + dreadnaughtBonus
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}

			warrior.revengeProcAura.Deactivate(sim)
		},
	})
}
