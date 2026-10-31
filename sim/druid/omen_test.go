package druid

import (
	"testing"
	"time"
)

// A spell's Omen of Clarity chance is the melee rate of 2 per minute taken
// over its cast time, an instant counting as the 1.5 s global cooldown.
func TestOmenSpellProcChanceFollowsCastTime(t *testing.T) {
	cases := []struct {
		castTime time.Duration
		want     float64
	}{
		{0, 0.05},
		{1500 * time.Millisecond, 0.05},
		{3 * time.Second, 0.10},
		{3500 * time.Millisecond, 2.0 * 3.5 / 60},
	}
	for _, c := range cases {
		if got := omenSpellProcChance(c.castTime, false); !closeTo(got, c.want) {
			t.Errorf("chance at %v = %v, want %v", c.castTime, got, c.want)
		}
	}
}

// Moonkin Form's effects 4 and 5 (spell 24858): +100% to Omen of Clarity's
// chance (SpellModOp 18) and -50% to its proc cooldown (SpellModOp 38).
func TestMoonkinFormDoublesTheChanceAndHalvesTheCooldown(t *testing.T) {
	if got := omenSpellProcChance(3*time.Second, true); !closeTo(got, 0.20) {
		t.Errorf("Moonkin chance for a 3 s cast = %v, want 0.20", got)
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
