package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// stormstrikeBaseManaCostPercent is SpellPower.PowerCostPct for spell 17364
// in client 1.60.1.70425, as the fraction ManaCostOptions.BaseCost takes.
const stormstrikeBaseManaCostPercent = 0.08

func (shaman *Shaman) registerStormstrikeSpell() {
	if !shaman.Talents.Stormstrike {
		return
	}

	stormStrikeAuras := shaman.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.StormstrikeAura(target)
	})

	shaman.Stormstrike = shaman.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_ShamanStormstrike,
		ClassSpellMask: ShamanSpellMaskStormstrike,
		ActionID:       core.ActionID{SpellID: 17364},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          SpellFlagShaman | core.SpellFlagAPL | core.SpellFlagMeleeMetrics,

		RequiredLevel: 40,

		// FOREVER: spellconst/shaman.json spell 17364 prices Stormstrike
		// at 8% of base mana since client 1.60.1.70425 (SpellPower
		// PowerCostPct 8; the 1.60.1.70374 client carried a flat 125).
		ManaCost: core.ManaCostOptions{
			BaseCost: stormstrikeBaseManaCostPercent,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    shaman.NewTimer(),
				Duration: time.Second * 8,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := shaman.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				stormStrikeAuras.Get(target).Activate(sim)
			}
		},
	})
}
