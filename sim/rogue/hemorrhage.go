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

	var hemoAuras core.AuraArray
	hemoAuras = rogue.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.HemorrhageAura(target)
	})

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

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			baseDamage := spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
				if len(hemoAuras) > 0 {
					hemoAura := hemoAuras.Get(target)
					hemoAura.Activate(sim)
					hemoAura.SetStacks(sim, 30)
				}
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
