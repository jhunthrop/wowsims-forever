package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (warrior *Warrior) registerSlamSpell() {
	rank := rankAtLevel(SlamLevel[:], warrior.Level)
	requiredLevel := SlamLevel[rank]
	// The engine keeps spell 11605, the id the UI and the preset
	// rotations name; the generated SlamSpellId[5] is Forever's reissue
	// 1310200. Same rank, same 87 damage - which id ships is the data
	// lane's call.
	spellID := int32(11605)
	flatDamageBonus := SlamBaseDamage[rank][0]

	warrior.Slam = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorSlam,
		ClassSpellMask: WarriorSpellMaskSlam,
		ActionID:       core.ActionID{SpellID: spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: requiredLevel,
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   rageCost(SlamManaCost[rank]),
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond*time.Duration(SlamCastTime[rank]) - time.Millisecond*100*time.Duration(warrior.Talents.ImprovedSlam),
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				if spell.CastTime() > 0 {
					warrior.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime+cast.CastTime, true)
				}
			},
			// The client gives Slam a 15 s category cooldown, and it
			// gives it to 11605 - the very id this file keeps - not
			// only to the reissue the dedup preferred. The engine had
			// no cooldown here at all; the generated value wins.
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: time.Duration(SlamCooldownMS[rank]) * time.Millisecond,
			},
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		FlatThreatBonus:  140, // Should this be 54 or the old 140 value from before SoD?
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := flatDamageBonus + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
