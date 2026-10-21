package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (rogue *Rogue) registerHemorrhageSpell() {
	if !rogue.Talents.Hemorrhage {
		return
	}

	// 16511 is Forever's one and only Hemorrhage rank
	// (data/builds/1.60.1.70009/spellranks.json, level 30); 17348 -
	// what stood here before - is a different, later-patch rank this
	// client does not carry, so the rotation's castSpell (and every
	// ExtraCastCondition/auraIsActive check keyed to it) could never
	// find this spell.
	spellID := int32(16511)

	actionID := core.ActionID{SpellID: spellID}

	// The client's Hemorrhage is a Rupture amplifier, not the vanilla
	// "+7 physical damage on the next 30 hits" (core.HemorrhageAura), so
	// the rogue keeps its own debuff; see RuptureDamageTakenMultiplier.
	hemoAuras := rogue.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    "Hemorrhage (Rupture)",
			ActionID: actionID,
			Duration: hemorrhageDebuffDuration,
		})
	})
	rogue.hemorrhageAuras = hemoAuras

	rogue.Hemorrhage = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueHemorrhage,
		ActionID:      actionID,
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         rogue.builderFlags(),
		RequiredLevel: 30,

		// The bleed debuff lives on the target, not the caster; see
		// priest/vampiric_embrace.go's RelatedSelfBuff comment for why
		// that field is used here anyway.
		RelatedSelfBuff: hemoAuras.Get(rogue.CurrentTarget),

		EnergyCost: core.EnergyCostOptions{
			Cost:   35.0,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},

		CritDamageBonus: rogue.lethality(),

		DamageMultiplier: rogue.mainHandStrikePct(hemorrhageWeaponDamagePct, hemorrhageDaggerWeaponDamagePct),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			baseDamage := spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
				hemoAuras.Get(target).Activate(sim)
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}

// RuptureDamageTakenMultiplier is the factor the target's Hemorrhage debuff
// puts on this rogue's Rupture damage (1 when it is not up).
func (rogue *Rogue) RuptureDamageTakenMultiplier(target *core.Unit) float64 {
	if rogue.hemorrhageAuras != nil && rogue.hemorrhageAuras.Get(target).IsActive() {
		return 1 + hemorrhageRuptureDamageBonus
	}
	return 1
}
