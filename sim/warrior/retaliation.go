package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	retaliationSpellID       = 20230
	retaliationStrikeSpellID = 20240
	retaliationLevel         = 20
	// retaliationDuration and retaliationCooldown are the client's 15
	// seconds and 15 minutes (spell 20230).
	retaliationDuration = 15 * time.Second
	retaliationCooldown = 15 * time.Minute
)

// registerRetaliationSpell is Retaliation: while it lasts, every melee
// attack that lands on the warrior is answered with a free swing of the
// main hand (spell 20240, one weapon-damage effect at 100%). The strike
// is a plain white-style attack and cannot itself trigger Retaliation.
func (warrior *Warrior) registerRetaliationSpell() {
	if warrior.Level < retaliationLevel {
		return
	}

	strike := warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID:    core.ActionID{SpellID: retaliationStrikeSpellID},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
		},
	})

	aura := warrior.RegisterAura(core.Aura{
		Label:    "Retaliation",
		ActionID: core.ActionID{SpellID: retaliationSpellID},
		Duration: retaliationDuration,
		OnSpellHitTaken: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.Landed() && spell.ProcMask.Matches(core.ProcMaskMelee) {
				strike.Cast(sim, spell.Unit)
			}
		},
	})

	warrior.Retaliation = warrior.RegisterSpell(BattleStance, core.SpellConfig{
		ActionID:    core.ActionID{SpellID: retaliationSpellID},
		SpellSchool: core.SpellSchoolPhysical,
		Flags:       core.SpellFlagAPL | core.SpellFlagHelpful,

		RequiredLevel: retaliationLevel,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: retaliationCooldown,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			aura.Activate(sim)
		},

		RelatedSelfBuff: aura,
	})
}
