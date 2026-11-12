package druid

import (
	"math"
	"testing"
)

// Rake's hit is the paw's weapon damage plus the rank's roll, and each tick
// the rank's amount plus 8% of attack power: Blizzard's 3 November 2026
// line ("Rake: Base points and attack power scaling increased by about
// 25%") as the first Forever cat logs measured it (rake.go).
func TestRakeHitIsWeaponDamagePlusTheRanksRoll(t *testing.T) {
	if got, want := rakeHitDamage(76, 150), 226.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("Rake hit with a 150 paw and a 76 roll = %v, want %v", got, want)
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
