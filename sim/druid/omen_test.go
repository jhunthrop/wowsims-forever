package druid

import (
	"testing"
	"time"
)

// A spell's Omen of Clarity chance is a flat 4% a cast, whatever its cast
// time. Beta logs (foreverlogs.gg, 34 caster druids, 12,402 casts, 547
// Clearcasting gains, 2026-09 to 2026-10) show no rise for cast-time spells
// over instants and about 4% a spell once melee procs are taken out; the
// only stated number (Reddit, 2026-09-30) is "4% chance on spells".
func TestOmenSpellProcChanceIsFlat(t *testing.T) {
	if got := omenSpellProcChance(false); !closeTo(got, 0.04) {
		t.Errorf("chance = %v, want 0.04", got)
	}
}

// Moonkin Form's effects 4 and 5 (spell 24858): +100% to Omen of Clarity's
// chance (SpellModOp 18) and -50% to its proc cooldown (SpellModOp 38).
func TestMoonkinFormDoublesTheChanceAndHalvesTheCooldown(t *testing.T) {
	if got := omenSpellProcChance(true); !closeTo(got, 0.08) {
		t.Errorf("Moonkin chance = %v, want 0.08", got)
	}
	if got := omenProcCooldown(false); got != 10*time.Second {
		t.Errorf("cooldown = %v, want 10 s", got)
	}
	if got := omenProcCooldown(true); got != 5*time.Second {
		t.Errorf("Moonkin cooldown = %v, want 5 s", got)
	}
}

func closeTo(got, want float64) bool {
	diff := got - want
	return diff < 1e-9 && diff > -1e-9
}
