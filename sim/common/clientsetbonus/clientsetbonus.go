// Package clientsetbonus reads the amount of a modelled client set bonus
// from its spell row, so a set model never types a number the client
// states. Every function panics on a row of another shape: the models name
// their bonus spells from the set rows, so a mismatch is a stale model.
package clientsetbonus

import (
	"fmt"

	"github.com/wowsims/classic/sim/core"
)

// auraAddFlatModifier is the client's flat spell-modifier aura.
const auraAddFlatModifier = core.ClientAuraAddFlatModifier

// auraDummy is the client's dummy aura, which carries a number the spell's
// script reads (a percentage for a set bonus that changes a spell's yield).
const auraDummy int32 = 4

const percentDivisor = 100.0

func onlyEffect(spellID int32) core.ClientEffect {
	spell := core.MustClientSpellRow(spellID)
	if len(spell.Effects) != 1 {
		panic(fmt.Sprintf("clientsetbonus: spell %d (%s) has %d effects, want 1", spellID, spell.Name, len(spell.Effects)))
	}
	return spell.Effects[0]
}

// DummyPercent is a bonus spell's dummy-aura number as a fraction (20 is
// 0.2).
func DummyPercent(spellID int32) float64 {
	effect := onlyEffect(spellID)
	if effect.Aura != auraDummy {
		panic(fmt.Sprintf("clientsetbonus: spell %d is aura %d, want the dummy aura %d", spellID, effect.Aura, auraDummy))
	}
	return effect.Points / percentDivisor
}
