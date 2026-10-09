package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// shadowMasteryTalents is Siphon Life (Affliction field 14) and Shadow
// Mastery 5/5 (field 16).
const shadowMasteryTalents = "0000000000000" + "1" + "0" + "5"

// TestShadowMasteryCoversEveryMaskedSpell pins spell 18271's class masks:
// each covered Shadow spell takes +1% a rank (5% at 5/5) as one multiplier,
// and Curse of Doom, outside the masks, takes none.
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
	}
	for name, spell := range covered {
		if got := spell.DamageMultiplierAdditive; math.Abs(got-1.05) > 1e-9 {
			t.Errorf("%s: additive multiplier %v with Shadow Mastery 5/5, want 1.05", name, got)
		}
	}
	if got := talented.CurseOfDoom.DamageMultiplierAdditive; got != 1 {
		t.Errorf("Curse of Doom is outside Shadow Mastery's masks but its additive multiplier is %v", got)
	}
}
