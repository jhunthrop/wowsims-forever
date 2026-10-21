package core

import (
	"math"
	"testing"
)

// Forever's normalized rage: a landed auto-attack pays weapon speed x a
// per-hand constant, half of that for an off-hand swing, and double on a
// critical strike; the damage dealt never enters. The constants and their
// provenance are on normalizedSwingRage.
func TestNormalizedSwingRage(t *testing.T) {
	oneHand := &Weapon{SwingSpeed: 2.6}
	twoHand := &Weapon{SwingSpeed: 3.8, TwoHanded: true}
	dagger := &Weapon{SwingSpeed: 1.8}

	cases := []struct {
		name    string
		weapon  *Weapon
		offHand bool
		crit    bool
		want    float64
	}{
		{"one-hand main hand", oneHand, false, false, 2.6 * 3.46},
		{"one-hand main hand crit", oneHand, false, true, 2.6 * 3.46 * 2},
		{"two-hand", twoHand, false, false, 3.8 * 4.50},
		{"two-hand crit", twoHand, false, true, 3.8 * 4.50 * 2},
		{"off-hand dagger", dagger, true, false, 1.8 * 3.46 * 0.5},
		{"off-hand dagger crit", dagger, true, true, 1.8 * 3.46 * 0.5 * 2},
		{"unarmed", &Weapon{}, false, false, 0},
		{"no weapon", nil, false, false, 0},
	}
	for _, c := range cases {
		if got := normalizedSwingRage(c.weapon, c.offHand, c.crit); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: rage = %v, want %v", c.name, got, c.want)
		}
	}
}

// A heavier weapon of the same speed generates the same rage: the
// normalization removes the damage term that let gear compound rage.
func TestNormalizedSwingRageIgnoresWeaponDamage(t *testing.T) {
	grey := &Weapon{SwingSpeed: 3.6, BaseDamageMin: 10, BaseDamageMax: 20, TwoHanded: true}
	epic := &Weapon{SwingSpeed: 3.6, BaseDamageMin: 200, BaseDamageMax: 300, TwoHanded: true}
	if a, b := normalizedSwingRage(grey, false, false), normalizedSwingRage(epic, false, false); a != b {
		t.Errorf("rage differs by weapon damage: grey %v, epic %v", a, b)
	}
}
