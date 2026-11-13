package druid

import (
	"math"
	"testing"
)

// Rake's hit is the rank's amount plus the same attack power share as a
// tick, and each tick the rank's amount plus 8% of attack power: Blizzard's
// 3 November 2026 line ("Rake: Base points and attack power scaling
// increased by about 25%") as the Forever cat logs measure it (rake.go). The
// hit carries no weapon damage: our advanced logs give every non-crit hit of
// one cat the same amount (61 on 20 hits while its white swings rolled 56 to
// 69 before armor), and the hit stays the rank's 7 above the tick as attack
// power moves.
func TestRakeHitIsTheRanksAmountPlusTheTicksAttackPowerShare(t *testing.T) {
	if got, want := rakeHitDamage(76, 1000), 156.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("Rake hit with a 76 roll at 1000 attack power = %v, want %v", got, want)
	}
	if got, want := rakeHitDamage(76, 1000)-rakeTickDamage(42, 1000), 34.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("Rake hit less a tick at the same attack power = %v, want the ranks' difference %v", got, want)
	}
}

func TestRakeTicksTakeEightPercentOfAttackPower(t *testing.T) {
	if got, want := rakeTickAttackPowerShare, 0.08; got != want {
		t.Fatalf("rakeTickAttackPowerShare = %v, want %v", got, want)
	}
	if got, want := rakeTickDamage(42, 1000), 122.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("Rake tick at 1000 attack power and a 42 amount = %v, want %v", got, want)
	}
}
