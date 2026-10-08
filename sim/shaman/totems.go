package shaman

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// totemGCD is every totem-drop spell's GCD per the client's own
// spellconst data (gcd_ms: 1000 on every real totem entry, e.g. Strength
// of Earth Totem id 8075, Searing Totem id 3599, Magma Totem id 8190) -
// shorter than core.GCDDefault's standard 1.5s, which the client reserves
// for non-totem casts.
const totemGCD = time.Second

func (shaman *Shaman) newTotemSpellConfig(flatCost float64, spellID int32) core.SpellConfig {
	return core.SpellConfig{
		ActionID: core.ActionID{SpellID: spellID},
		Flags:    SpellFlagShaman | SpellFlagTotem | core.SpellFlagAPL,

		ManaCost: core.ManaCostOptions{
			FlatCost:   flatCost,
			Multiplier: shaman.totemManaMultiplier(),
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: totemGCD,
			},
		},
	}
}

// standingTotemDuration is the client's duration_ms (300000) on every
// earth, air and water totem rank; the fire totems carry their own
// per-rank lives (Searing 30-55s, Magma 20s). Writing a totem's life
// anywhere else than here and in the fire totems' tables is how the
// engine drifted to the vanilla two minutes.
const standingTotemDuration = 5 * time.Minute

// newTotemLifetimeConfig is the aura that stands for a dropped totem: it
// is on the shaman from the cast until the totem's life ends or another
// totem takes the slot. The spell links it as RelatedSelfBuff, so the
// totem's life is read from this one aura wherever a totem is described.
// It carries no ActionID: the buff a totem grants is found by the
// rotation under the buff's own id, and a second aura under the totem's
// id would answer that lookup instead.
func newTotemLifetimeConfig(name string, rank int) core.Aura {
	label := name
	if rank > 0 {
		label = fmt.Sprintf("%s (Rank %d)", name, rank)
	}
	return core.Aura{
		Label:    label,
		Duration: standingTotemDuration,
	}
}

// registerBuffTotemLifetime registers the lifetime aura of a totem whose
// effect is a shared buff aura (core.StrengthOfEarthTotemAura and its
// siblings run for the vanilla two minutes, so the buff is tied to the
// totem's own life). A buff the raid-buff build made permanent keeps its
// own expiry, and still ends when the totem is replaced, as it always did
// (the rotations rely on that: they cast one air totem after the other).
func (shaman *Shaman) registerBuffTotemLifetime(name string, rank int, buff *core.Aura) *core.Aura {
	config := newTotemLifetimeConfig(name, rank)
	config.OnGain = func(aura *core.Aura, sim *core.Simulation) {
		buff.Activate(sim)
		if buff.Duration != core.NeverExpires {
			buff.UpdateExpires(sim, aura.ExpiresAt())
		}
	}
	config.OnExpire = func(_ *core.Aura, sim *core.Simulation) {
		buff.Deactivate(sim)
	}
	return shaman.RegisterAura(config)
}

// dropStandingTotem makes lifetime the totem standing in slot: the totem
// it replaces ends first, and the slot's expiry is read from lifetime.
func (shaman *Shaman) dropStandingTotem(sim *core.Simulation, slot int, spell *core.Spell, lifetime *core.Aura) {
	if previous := shaman.ActiveTotemBuffs[slot]; previous != nil {
		previous.Deactivate(sim)
	}
	shaman.TotemExpirations[slot] = sim.CurrentTime + lifetime.Duration
	shaman.ActiveTotems[slot] = spell
	shaman.ActiveTotemBuffs[slot] = lifetime
	lifetime.Activate(sim)
}
