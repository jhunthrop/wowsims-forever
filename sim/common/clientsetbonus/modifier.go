package clientsetbonus

import (
	"fmt"

	"github.com/wowsims/classic/sim/core"
)

// The client's SpellModOp values (the effect's EffectMiscValue_0) that
// core.EquipSpellMod does not express as a spell mod, for the set models
// that apply them by hand.
const (
	ModOpThreat     int32 = 2
	ModOpAllEffects int32 = 8
	ModOpDot        int32 = 22
)

// PercentModifier is the percentage a bonus spell's percent-modifier
// effect (client aura 108) gives its property op, as a fraction (-25 is
// -0.25). Panics when the spell has no such effect, so a stale model stops
// the engine at init.
func PercentModifier(spellID, op int32) float64 {
	spell := core.MustClientSpellRow(spellID)
	for _, effect := range spell.Effects {
		if effect.Aura == core.ClientAuraAddPctModifier && effect.Misc0 == op {
			return effect.Points / percentDivisor
		}
	}
	panic(fmt.Sprintf("clientsetbonus: spell %d (%s) has no percent modifier of property %d", spellID, spell.Name, op))
}
