package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Whirlwind is the one ability in this package that reads NOTHING out
// of constants_auto_gen.go, and the third row the data lane owes a fix
// for. Column by column:
//
//   - WhirlwindManaCost[0] and WhirlwindCooldownMS[0] are both 0. Two
//     rows share the name, the rank (0) and spell_level 36 - the
//     250-tenths, 10 s 1680 and a free reissue, 462891 - and the dedup
//     kept the free one. The two constants below are therefore read by
//     hand from data/builds/1.60.1.69893/spellconst/warrior.json, which
//     gives 1680 cost 250 tenths and category_cooldown_ms 10000.
//   - WhirlwindSpellId[0] is 462891 for the same reason, where the UI
//     and the pinned rotation both name 1680. Ids are not swapped here.
//   - WhirlwindBaseDamage is {0, 0}, and that IS sound: the client says
//     Whirlwind is pure weapon damage with no flat term. There is
//     simply nothing to read, which is why ApplyEffects adds no base
//     damage rather than adding a generated zero.
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
