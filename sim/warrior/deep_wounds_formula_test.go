package warrior

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Deep Wounds, as Blizzard confirmed on forever-bugs #234 (2026-10-05):
// "Rolls over its damage when refreshed", "Doesn't reset its tick timer
// when it is refreshed" and "Doesn't scale with Attack Power ... it scales
// with your weapon damage and not with your Attack Power at all". The
// talent text: 20/40/60% of the melee weapon's average damage over 12 sec.
func TestDeepWoundsIsWeaponDamageWithoutAttackPower(t *testing.T) {
	weapon := &core.Weapon{BaseDamageMin: 100, BaseDamageMax: 200, SwingSpeed: 3.6, AttackPowerPerDPS: core.DefaultAttackPowerPerDPS}
	for points, want := range map[int32]float64{1: 30, 2: 60, 3: 90} {
		if got := deepWoundsTotal(weapon, false, points); math.Abs(got-want) > 1e-9 {
			t.Errorf("main hand, %d points: Deep Wounds total %v, want %v (weapon damage only)", points, got, want)
		}
	}
	// The off hand keeps its half-damage penalty.
	if got, want := deepWoundsTotal(weapon, true, 3), 45.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("off hand, 3 points: Deep Wounds total %v, want %v", got, want)
	}
}

// A new crit adds its bleed to what the running one has left and spreads
// the sum over the four ticks again.
func TestDeepWoundsRollsItsDamageOver(t *testing.T) {
	// 25 a tick with 3 ticks left (75) plus a new 100: 175 over 4 ticks.
	if got, want := deepWoundsRolledTick(25, 3, 100), 43.75; math.Abs(got-want) > 1e-9 {
		t.Errorf("rolled tick = %v, want %v", got, want)
	}
	// Nothing running: the new bleed alone.
	if got, want := deepWoundsRolledTick(0, 0, 100), 25.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("fresh tick = %v, want %v", got, want)
	}
}
