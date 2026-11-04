package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Venom is an Assassination talent (node 105712, tier 4, one rank). In
// 70755 it is a passive, 1329328 ("Causes your Shank ability to increase
// your Poison damage by 30% and your chance to apply Poisons by 20% for 6
// sec"): e0 aura 4, bp 100, triggers 1310703 on Shank. 1310703 is no longer
// a finisher (its cost, cooldown and combo-point durations were deleted):
// DurationIndex 32 (6 sec), e0 aura 108 bp 30 on mask 8192/8 (Instant
// Poison), e1 aura 108 bp 30 misc 22 (periodic) on mask 65536 (Deadly
// Poison), e2 aura 107 misc 18 (proc chance) bp 20 on every poison's mask
// (268558336). The aura 108 modifiers share Vile Poisons' (16513) kind,
// misc value and masks, so the two add, never multiply.
const (
	venomPoisonDamageBonus     = 0.30
	venomPoisonProcChanceBonus = 0.20
	venomDuration              = 6 * time.Second
	venomBuffSpellID           = 1310703
)

// venomDamageScale is the factor that takes the poisons' current
// Vile-Poisons multiplier (1 + vile) to (1 + vile + venom).
func (rogue *Rogue) venomDamageScale() float64 {
	vile := rogue.getPoisonDamageMultiplier()
	return (vile + venomPoisonDamageBonus) / vile
}

// scalePoisonDamage multiplies the damage multiplier of the poisons on
// Venom's two masks, Instant (8192) and Deadly (65536). Wound Poison's mask
// (268435456) is outside them.
func (rogue *Rogue) scalePoisonDamage(factor float64) {
	for _, spell := range []*core.Spell{rogue.InstantPoison, rogue.deadlyPoisonTick} {
		if spell != nil {
			spell.DamageMultiplier *= factor
		}
	}
}

// registerVenomAura registers Venom's 1310703 buff for Shank to apply, or
// returns nil without the talent.
func (rogue *Rogue) registerVenomAura() *core.Aura {
	if !rogue.Talents.Venom {
		return nil
	}

	rogue.VenomAura = rogue.RegisterAura(core.Aura{
		Label:    "Venom",
		ActionID: core.ActionID{SpellID: venomBuffSpellID},
		Duration: venomDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.additivePoisonBonusChance += venomPoisonProcChanceBonus
			rogue.scalePoisonDamage(rogue.venomDamageScale())
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.additivePoisonBonusChance -= venomPoisonProcChanceBonus
			rogue.scalePoisonDamage(1 / rogue.venomDamageScale())
		},
	})
	return rogue.VenomAura
}
