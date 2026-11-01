package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// shadowMasteryTalents is Siphon Life (Affliction field 14), Shadow Mastery
// 5/5 (field 16), Wrack (field 17) and Shadowburn (Destruction field 8).
const shadowMasteryTalents = "0000000000000" + "1" + "0" + "5" + "1" + "--" + "00000001"

// TestShadowMasteryCoversEveryMaskedSpell pins spell 18271's class masks
// (70707): effect 0 (direct damage) on (0x80093, 0, 0) and effect 1 (periodic
// damage) on (0x440A, 0x40003, 0). Each covered Shadow spell takes +2% a rank
// (10% at 5/5, 1.60.1.70425) as one multiplier. Bane of Doom (603, mask word 1
// 0x2) and Wrack (1316697, word 1 0x40000) sit inside effect 1's mask.
func TestShadowMasteryCoversEveryMaskedSpell(t *testing.T) {
	_, talented, _ := newWarlockForDamageTest(t, shadowMasteryTalents)

	last := func(spells []*core.Spell) *core.Spell { return spells[len(spells)-1] }
	covered := map[string]*core.Spell{
		"Shadow Bolt":    last(talented.ShadowBolt),
		"Corruption":     last(talented.Corruption),
		"Curse of Agony": last(talented.CurseOfAgony),
		"Death Coil":     last(talented.DeathCoil),
		"Drain Life":     last(talented.DrainLife),
		"Drain Soul":     last(talented.DrainSoul),
		"Siphon Life":    last(talented.SiphonLife),
		"Bane of Doom":   talented.CurseOfDoom,
		"Wrack":          talented.Wrack,
		"Shadowburn":     last(talented.Shadowburn),
	}
	for name, spell := range covered {
		if got := spell.DamageMultiplierAdditive; math.Abs(got-1.10) > 1e-9 {
			t.Errorf("%s: additive multiplier %v with Shadow Mastery 5/5, want 1.10", name, got)
		}
	}
}
