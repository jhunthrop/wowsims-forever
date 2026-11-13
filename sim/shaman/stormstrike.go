package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Stormstrike is the client's 17364 after Blizzard's 10 November hotfix
// (t/2403040, Enhancement: "Stormstrike damage reduced to 100% weapon damage
// (was 150%), and cooldown increased to 8 seconds (was 6 seconds)"), which
// push 113722 states in the 1.60.1.70938 data: SpellCooldowns 55241
// RecoveryTime 8000 and SpellEffect 1368308 (effect 2, Effect 31, weapon
// percent damage) 100. The cooldown and the 8% base mana cost read the
// generated constants. The weapon percent scales effect 0's normalized
// weapon damage; the generated arrays carry no effect-2 column, so
// TestStormstrikeIsTheClientRow checks it against the vendored client row.
const stormstrikeWeaponDamagePercent = 1.0

func stormstrikeCooldown() time.Duration {
	return time.Duration(StormstrikeCooldownMS[0]) * time.Millisecond
}

// stormstrikeBaseManaCost is SpellPower.PowerCostPct for 17364 as the
// fraction ManaCostOptions.BaseCost takes.
func stormstrikeBaseManaCost() float64 {
	return StormstrikeManaCostPct[0] / 100
}

func (shaman *Shaman) registerStormstrikeSpell() {
	if !shaman.Talents.Stormstrike {
		return
	}

	stormStrikeAuras := shaman.newStormstrikeChargeAuras()

	shaman.Stormstrike = shaman.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_ShamanStormstrike,
		ClassSpellMask: ShamanSpellMaskStormstrike,
		ActionID:       core.ActionID{SpellID: 17364},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          SpellFlagShaman | core.SpellFlagAPL | core.SpellFlagMeleeMetrics,

		RequiredLevel: 40,

		ManaCost: core.ManaCostOptions{
			BaseCost: stormstrikeBaseManaCost(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    shaman.NewTimer(),
				Duration: stormstrikeCooldown(),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := shaman.stormstrikeWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				stormStrikeAuras.Get(target).Activate(sim)
			}
		},
	})
}

// Stormstrike's debuff (17364 effect 1): aura 271, damage taken from the
// caster +20% for the spells in SpellClassMask_0 0x100003 (Lightning Bolt,
// Chain Lightning and Earth Shock), with ProcCharges 1 (SpellAuraOptions
// 129538) and a 12 s duration, in client 1.60.1.70707. Wowhead Forever:
// "increase the damage you deal to the target with your next Lightning
// Bolt, Chain Lightning, or Earth Shock spell by 20% for 12 sec". So the
// first of those spells to hit takes the 20% and spends the charge; the
// rest of the Nature school is untouched (Era's debuff raised all Nature
// damage taken for two charges).
// stormstrikeWeaponDamage is Stormstrike's strike: client 17364 effect 0 is
// effect 121, normalized weapon damage (1.60.1.70707 SpellEffect), so attack
// power counts at the weapon type's normalized speed, not its swing speed,
// and effect 2 scales it (100% since the 10 November hotfix).
func (shaman *Shaman) stormstrikeWeaponDamage(sim *core.Simulation, attackPower float64) float64 {
	return stormstrikeWeaponDamagePercent * shaman.MHNormalizedWeaponDamage(sim, attackPower)
}

const (
	stormstrikeChargeBonus    = 1.20
	stormstrikeChargeDuration = 12 * time.Second
	stormstrikeChargeMask     = ShamanSpellMaskLightningBolt | ShamanSpellMaskChainLightning | ShamanSpellMaskEarthShock
)

// newStormstrikeChargeAuras is the debuff on each target. The bonus is a
// spell mod on the shaman while the debuff is up on the target being
// struck; the published fight has one target.
func (shaman *Shaman) newStormstrikeChargeAuras() core.AuraArray {
	bonus := shaman.AddDynamicMod(core.SpellModConfig{
		Kind:       core.SpellMod_DamageDone_Pct,
		ClassMask:  stormstrikeChargeMask,
		FloatValue: stormstrikeChargeBonus,
	})
	auras := shaman.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    "Stormstrike-" + shaman.Label,
			ActionID: core.ActionID{SpellID: 17364},
			Duration: stormstrikeChargeDuration,
			OnGain: func(_ *core.Aura, _ *core.Simulation) {
				bonus.Activate()
			},
			OnExpire: func(_ *core.Aura, _ *core.Simulation) {
				bonus.Deactivate()
			},
		})
	})
	core.MakePermanent(shaman.RegisterAura(core.Aura{
		Label: "Stormstrike Charge Spender",
		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.Matches(stormstrikeChargeMask) || !result.Landed() {
				return
			}
			if debuff := auras.Get(result.Target); debuff.IsActive() {
				debuff.Deactivate(sim)
			}
		},
	}))
	return auras
}
