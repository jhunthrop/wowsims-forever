package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Revenge is learned at levels 14 through 54 from the trainer and its rank
// 6 (25288, level 60) is an Ahn'Qiraj book rank, so a level-60 character
// casts rank 5 while core.IncludeAQ is off (the spellranks table marks 25288
// "book"). A character below the first rank still registers rank 1, which
// the level ladder's rotation rewrite never casts.
//
// Its threat is UNCONFIRMED: the client row carries a single school-damage
// effect and no threat effect, so the fork's long-standing figures stay
// (the damage times 2.25 plus a flat 2.25 x 2 x the spell's level, the
// Sunder Armor formula that predates the client's own Sunder numbers).
const revengeThreatMultiplier = 2.25

func (warrior *Warrior) registerRevengeSpell(cdTimer *core.Timer) {
	topTrainerRank := core.TernaryInt(core.IncludeAQ, RevengeRanks, RevengeRanks-1)
	rank := max(1, min(rankAtLevel(RevengeLevel[:], warrior.Level), topTrainerRank))
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

		RequiredLevel: RevengeLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   rageCost(RevengeManaCost[rank]),
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: time.Duration(RevengeCooldownMS[rank]) * time.Millisecond,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.revengeProcAura.IsActive()
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: revengeThreatMultiplier,
		FlatThreatBonus:  revengeThreatMultiplier * 2 * revengeLevel,
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
