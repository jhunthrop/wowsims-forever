package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (warrior *Warrior) registerOverpowerSpell(cdTimer *core.Timer) {
	rank := rankAtLevel(OverpowerLevel[:], warrior.Level)
	bonusDamage := OverpowerBaseDamage[rank][0]
	spellID := OverpowerSpellId[rank]

	castConfig := core.CastConfig{
		DefaultCast: core.Cast{
			GCD: core.GCDDefault,
		},
		IgnoreHaste: true,
	}
	// Rank 0 - a warrior below the lowest real OverpowerLevel entry -
	// carries a zero cooldown; guard as slam.go does.
	if cooldownMS := OverpowerCooldownMS[rank]; cooldownMS > 0 {
		castConfig.CD = core.Cooldown{
			Timer:    cdTimer,
			Duration: time.Duration(cooldownMS) * time.Millisecond,
		}
	}

	warrior.RegisterAura(core.Aura{
		Label:    "Overpower Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidDodge() {
				warrior.OverpowerAura.Activate(sim)
			}
		},
	})

	warrior.OverpowerAura = warrior.RegisterAura(core.Aura{
		Label:    "Overpower Aura",
		ActionID: core.ActionID{SpellID: spellID},
		Duration: time.Second * 5,
	})

	warrior.Overpower = warrior.RegisterSpell(BattleStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorOverpower,
		ClassSpellMask: WarriorSpellMaskOverpower,
		ActionID:       core.ActionID{SpellID: spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: OverpowerLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   rageCost(OverpowerManaCost[rank]),
			Refund: 0.8,
		},
		Cast: castConfig,
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			// Bloodthrill opens the same window off a separate 6s aura
			// (bloodthrill.go), since it is not the vanilla 5s dodge
			// proc; nil when Bloodthrill is not talented.
			return warrior.OverpowerAura.IsActive() || (warrior.BloodthrillAura != nil && warrior.BloodthrillAura.IsActive())
		},

		BonusCritRating: 25 * core.CritRatingPerCritChance * float64(warrior.Talents.ImprovedOverpower),

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 0.75,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := bonusDamage + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialNoBlockDodgeParry)

			warrior.OverpowerAura.Deactivate(sim)
			if warrior.BloodthrillAura != nil {
				warrior.BloodthrillAura.Deactivate(sim)
			}
			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
