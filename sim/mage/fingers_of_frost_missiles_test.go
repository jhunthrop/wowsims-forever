package mage

import (
	"github.com/wowsims/classic/sim/core"
	"testing"
)

// The published level-60 raid Frost build, which carries Precision 3/3 and
// Fingers of Frost 2/2.
const round10FrostRaidTalents = "230005200000000000-51020000000000000-253510130101301221"

// Fingers of Frost and Shatter reach "a direct-damage cast of the mage's
// own" (design/reviews/2026-10-07-frost-fingers-of-frost.md): never an
// Arcane Missiles missile, which is a passive tick of a channel. The check
// asked for the mage flag OR the APL flag, so every missile spent a charge
// and took Shatter's crit.
func TestArcaneMissilesMissilesDoNotSpendFingersOfFrost(t *testing.T) {
	built, sim, boss := newFrostMageSimWithTalents(t, bossTarget(), round10FrostRaidTalents)
	var missile *core.Spell
	for _, spell := range built.Spellbook {
		if spell.SpellCode == SpellCode_MageArcaneMissilesTick {
			missile = spell
		}
	}
	if missile == nil {
		t.Fatal("no Arcane Missiles missile spell is registered")
	}
	if built.consumesFingersOfFrost(missile) {
		t.Error("an Arcane Missiles missile counts as a Fingers of Frost cast")
	}
	built.grantFingersOfFrost(sim)
	missile.ApplyEffects(sim, boss, missile)
	if got := built.FingersOfFrostAura.GetStacks(); got != 2 {
		t.Errorf("after one missile %d Fingers of Frost charges remain, want 2", got)
	}
}
