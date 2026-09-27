package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (warrior *Warrior) registerPummelSpell() {
	rank := rankAtLevel(PummelLevel[:], warrior.Level)
	damage := PummelBaseDamage[rank][0]

	castConfig := core.CastConfig{
		DefaultCast: core.Cast{
			GCD: core.GCDDefault,
		},
		IgnoreHaste: true,
	}
	// Rank 0 - a warrior below the lowest real PummelLevel entry (20) -
	// carries a zero cooldown; guard as slam.go does.
	if cooldownMS := PummelCooldownMS[rank]; cooldownMS > 0 {
		castConfig.CD = core.Cooldown{
			Timer:    warrior.NewTimer(),
			Duration: time.Duration(cooldownMS) * time.Millisecond,
		}
	}

	warrior.RegisterSpell(BerserkerStance, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: PummelSpellId[rank]},
		ClassSpellMask: WarriorSpellMaskPummel,
		RequiredLevel:  PummelLevel[rank],
		Rank:           rank,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagBinary | SpellFlagOffensive,

		RageCost: core.RageCostOptions{
			Cost:   rageCost(PummelManaCost[rank]),
			Refund: 0.8,
		},
		Cast: castConfig,

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
