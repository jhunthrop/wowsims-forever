package shaman

import (
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
