package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// bloodthirstAttackPowerCoefficient is the "35% of your Attack Power" of
// the client's own rank description (Fury node 105930, rank spell
// 23881): "Instantly attack the target causing damage equal to 35% of
// your Attack Power plus 30 and increasing your movement speed by 10%
// for 10 sec."
//
// The percentage is effect 3 of spell 23894 in
// data/builds/1.60.1.69893/spellconst/warrior.json and reads 35 at every
// rank; the generator emits only the school-damage effect, so this one
// number is typed here with the client's text beside it rather than read
// from constants_auto_gen.go. The "plus 30" is rank 1's flat base -
// BloodthirstBaseDamage carries all four (30/37/43/48) and rank 4 is
// what a level-60 warrior casts.
//
// Vanilla's Bloodthirst was a flat 45% of attack power with no base
// term. That is a different spell, not a renamed one, so the 0.45 that
// stood here is gone.
const bloodthirstAttackPowerCoefficient = 0.35

func (warrior *Warrior) registerBloodthirstSpell(cdTimer *core.Timer) {
	if !warrior.Talents.Bloodthirst {
		return
	}

	rank := rankAtLevel(BloodthirstLevel[:], warrior.Level)
	baseDamage := BloodthirstBaseDamage[rank][0]

	warrior.Bloodthirst = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorBloodthirst,
		ClassSpellMask: WarriorSpellMaskBloodthirst,
		ActionID:       core.ActionID{SpellID: BloodthirstSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: BloodthirstLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   rageCost(BloodthirstManaCost[rank]),
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: time.Duration(BloodthirstCooldownMS[rank]) * time.Millisecond,
			},
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := baseDamage + bloodthirstAttackPowerCoefficient*spell.MeleeAttackPower(target)
			result := spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeSpecialHitAndCrit)
			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
