package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Venom (1310703) and Vile Poisons (16513) are aura 108 modifiers on the
// spell-class masks 8192 (Instant Poison) and 65536 (Deadly Poison). Wound
// Poison's mask is 268435456, outside both (SpellClassOptions, 1.60.1.70707;
// sim-audit round 3 R2), so neither talent may scale it.
func TestVenomAndVilePoisonsLeaveWoundPoisonAlone(t *testing.T) {
	instant := &core.Spell{DamageMultiplier: 1}
	deadly := &core.Spell{DamageMultiplier: 1}
	wound := &core.Spell{DamageMultiplier: 1}
	rogue := &Rogue{InstantPoison: instant, deadlyPoisonTick: deadly, WoundPoison: wound}

	rogue.scalePoisonDamage(1.3)

	if instant.DamageMultiplier != 1.3 || deadly.DamageMultiplier != 1.3 {
		t.Fatalf("Instant and Deadly Poison take Venom: got %v and %v, want 1.3", instant.DamageMultiplier, deadly.DamageMultiplier)
	}
	if wound.DamageMultiplier != 1 {
		t.Fatalf("Wound Poison is outside Venom's masks: got %v, want 1", wound.DamageMultiplier)
	}
}
