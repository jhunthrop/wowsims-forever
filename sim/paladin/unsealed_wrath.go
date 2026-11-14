package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// sealEffect is what a Seal does to a landed melee swing: its chance to
// activate and the effect itself. Each Seal aura's own hit handler and
// Unsealed Wrath's stored copies read the same one, so a stored Seal
// activates exactly as the Seal would.
type sealEffect struct {
	// family names the Seal (every rank of one Seal shares it): Unsealed
	// Wrath consumes its stacks when a swing lands under a different one.
	family string
	// chance rolls the Seal's activation chance; nil is "every swing".
	chance func(sim *core.Simulation, procMask core.ProcMask) bool
	// apply is the Seal's effect; nil for a Seal with no swing effect
	// (Seal of the Crusader).
	apply func(sim *core.Simulation, target *core.Unit)
}

// tryActivate gives the Seal its normal chance to activate on a swing.
func (effect *sealEffect) tryActivate(sim *core.Simulation, target *core.Unit, procMask core.ProcMask) {
	if effect.apply == nil {
		return
	}
	if effect.chance == nil || effect.chance(sim, procMask) {
		effect.apply(sim, target)
	}
}

// registerSealEffect records what the Seal aura does to a swing.
func (paladin *Paladin) registerSealEffect(aura *core.Aura, effect *sealEffect) {
	if paladin.sealEffects == nil {
		paladin.sealEffects = map[*core.Aura]*sealEffect{}
	}
	paladin.sealEffects[aura] = effect
}

// activeSealEffect is the swing effect of the Seal that is up, or nil.
func (paladin *Paladin) activeSealEffect() *sealEffect {
	if paladin.currentSeal == nil || !paladin.currentSeal.IsActive() {
		return nil
	}
	return paladin.sealEffects[paladin.currentSeal]
}

// isSealSwing reports whether a landed hit is a swing a Seal acts on.
func isSealSwing(spell *core.Spell, result *core.SpellResult) bool {
	return result.Landed() && spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit)
}

const (
	// Unsealed Wrath (node 105692, Retribution tier 6 col 1, spell
	// 1310735, client 1.60.1.70707): "Your melee attacks store a copy of
	// your active Seal for 15 sec, stacking up to 3 times. Landing an
	// attack with a different Seal consumes all stacks, giving the stored
	// Seal its normal chance to activate once per stack."
	//
	// Read as: a landed white swing (the swing every Seal acts on) under
	// the same Seal as the stored copies adds a stack and refreshes the 15
	// seconds; a landed swing under a different Seal first rolls the
	// stored Seal once per stack (its proc chance, without Seal of
	// Command's 1 second internal cooldown, which guards one Seal against
	// double swings rather than a stored copy) and then starts a new
	// stack of the Seal that is up. Blizzard's 11 November 2026 hotfix
	// (t/2403040): "Unsealed Fury no longer generates echoes from Windfury
	// Totem attacks", so an extra attack (Reckoning, Hand of Justice)
	// still releases stored copies, being a landed attack under whatever
	// Seal is up, but stores none. Since push 113887 a Windfury Totem
	// attack is a yellow strike (core.ApplyWindfury), not a white swing,
	// so it neither stores nor releases. Our launch logs (Echo of
	// Command 1311703, Echo of Righteousness 1311704, Echo of Fury 1311701,
	// three Retribution paladins at 33 to 45) agree: 62 of 62 stacks were
	// added by a white swing and none by any of 33 Holy Strikes, and the
	// copies share one 15 second timer that each new stack restarts (three
	// expiries 15.0 to 15.1 seconds after the last stack).
	unsealedWrathActionID  = 1310735
	unsealedWrathDuration  = 15 * time.Second
	unsealedWrathMaxStacks = 3
)

// registerUnsealedWrath is the Retribution capstone; it needs the Seals
// registered first.
func (paladin *Paladin) registerUnsealedWrath() {
	if !paladin.Talents.UnsealedWrath {
		return
	}

	var stored *sealEffect
	copies := paladin.RegisterAura(core.Aura{
		Label:     "Unsealed Wrath",
		ActionID:  core.ActionID{SpellID: unsealedWrathActionID},
		Duration:  unsealedWrathDuration,
		MaxStacks: unsealedWrathMaxStacks,
		OnExpire: func(*core.Aura, *core.Simulation) {
			stored = nil
		},
	})

	paladin.RegisterAura(core.Aura{
		Label:    "Unsealed Wrath Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !isSealSwing(spell, result) {
				return
			}
			active := paladin.activeSealEffect()
			if active == nil {
				return
			}
			if copies.IsActive() && stored.family != active.family {
				for range copies.GetStacks() {
					stored.tryActivate(sim, result.Target, spell.ProcMask)
				}
				copies.Deactivate(sim)
			}
			if spell.IsExtraAttack() {
				return
			}
			if !copies.IsActive() {
				copies.Activate(sim)
				stored = active
			} else {
				copies.Refresh(sim)
			}
			copies.AddStack(sim)
		},
	})
}
