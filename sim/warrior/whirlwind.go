package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Whirlwind's generated rows are the third the data lane owes a fix
// for. Two rows share the name, the rank (0) and spell_level 36 - the
// 250-tenths 1680 and a free reissue - and the dedup kept the free one,
// so WhirlwindManaCost[0] and WhirlwindCooldownMS[0] are both 0 where
// the client's own row says 250 tenths and a 10 s category cooldown.
// Both numbers below are therefore read by hand from
// data/builds/1.60.1.69893/spellconst/warrior.json.
//
// WhirlwindBaseDamage IS sound and IS read - it is {0, 0}, which is the
// client saying Whirlwind is pure weapon damage with no flat term, and
// that is why no base damage appears in ApplyEffects.
const (
	whirlwindRageCost = 25.0
	whirlwindCooldown = time.Second * 10
)

func (warrior *Warrior) registerWhirlwindSpell() {
	results := make([]*core.SpellResult, min(4, warrior.Env.GetNumTargets()))

	warrior.Whirlwind = warrior.RegisterSpell(BerserkerStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorWhirlwind,
		ClassSpellMask: WarriorSpellMaskWhirlwind,
		ActionID:       core.ActionID{SpellID: 1680},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagAPL | SpellFlagOffensive,

		RageCost: core.RageCostOptions{
			Cost: whirlwindRageCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: whirlwindCooldown,
			},
		},
		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1.25,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for idx := range results {
				baseDamage := spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
				results[idx] = spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}

			for _, result := range results {
				spell.DealDamage(sim, result)
			}
		},
	})
}
