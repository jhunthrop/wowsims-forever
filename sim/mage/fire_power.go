package mage

import (
	"github.com/wowsims/classic/sim/core"
)

// Fire Power (node 105782, three ranks since client 1.60.1.70425):
// "Increases the damage done by your Fire spells by 1/2/3% per your Fire
// effects on the target, up to a maximum increase of 4/8/12%. Effects
// which mix multiple schools of magic do not qualify."
//
// The mage's Fire effects on a target are its Fire-school periodic and
// debuff auras there: the Fireball, Pyroblast and Flamestrike dots and
// Ignite (each tagged mageFireEffectTag where its aura is built) plus the
// Improved Scorch debuff it keeps up. Frostfire Bolt's dot mixes schools
// and does not count. The bonus is read when a Fire spell resolves, so a
// dot it applies snapshots the count at that moment, the way the client
// snapshots a dot's modifiers on application.
const (
	mageFireEffectTag           = "MageFireEffect"
	firePowerDamagePerEffectPct = 0.01
	firePowerMaxEffects         = 4
)

// firePowerBonus is the additive damage share for one Fire spell cast at
// target, at the mage's Fire Power rank.
func (mage *Mage) firePowerBonus(target *core.Unit) float64 {
	rank := mage.Talents.FirePower
	if rank == 0 {
		return 0
	}
	effects := int(target.NumActiveAurasWithTag(mageFireEffectTag))
	if mage.ImprovedScorchAuras != nil && mage.ImprovedScorchAuras.Get(target).IsActive() {
		effects++
	}
	return firePowerDamagePerEffectPct * float64(rank) * float64(min(effects, firePowerMaxEffects))
}

func (mage *Mage) applyFirePower() {
	if mage.Talents.FirePower == 0 {
		return
	}

	// Every Fire damage spell registers after ApplyTalents, so each one is
	// wrapped as it registers: the bonus is added for the spell's own
	// resolution and taken back off after it, like Arcane Power's
	// temporary DamageMultiplierAdditive but per cast and per target.
	mage.OnSpellRegistered(func(spell *core.Spell) {
		if spell.ClassSpellMask&MageSpellMaskFireDamage == 0 || spell.ApplyEffects == nil {
			return
		}
		apply := spell.ApplyEffects
		spell.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			bonus := mage.firePowerBonus(target)
			spell.DamageMultiplierAdditive += bonus
			apply(sim, target, spell)
			spell.DamageMultiplierAdditive -= bonus
		}
	})
}
