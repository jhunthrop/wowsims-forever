package core

import (
	"math"
	"testing"
)

// Rage from damage taken divides by the rage conversion value at the
// receiver's own level: Kalgan's 1.x formula ("damage taken / rage
// conversion at your level x 2.5", 230.6 at 60) and cmangos-classic's
// Player::RewardRage, which takes GetLevel() of the player gaining the
// rage. A level-60 tank hit by a level-63 boss converts at 60's value.
func TestRageFromDamageTakenUsesTheReceiversLevel(t *testing.T) {
	tank := &Unit{Level: 60}
	boss := &Unit{Level: 63}
	if got, want := damageTakenRageConversion(tank, boss), GetRageConversion(60); got != want {
		t.Errorf("rage conversion for a level-60 tank hit by a level-63 boss = %.2f, want the tank's level-60 %.2f", got, want)
	}
}

// Blizzard's 8 October 2026 beta notes: rage from damage taken is balanced
// on an assumed armor "in the 20% to 40% range, varying by level".
func TestAssumedArmorReductionRunsFromTwentyToFortyPercent(t *testing.T) {
	for _, tc := range []struct {
		level int32
		want  float64
	}{{1, 0.20}, {60, 0.40}, {30, 0.20 + 0.20*29.0/59.0}, {63, 0.40}} {
		if got := assumedArmorReduction(tc.level); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("assumed armor at level %d = %v, want %v", tc.level, got, tc.want)
		}
	}
}

// "Rage gain from damage is calculated by ignoring Armor. So, wearing
// Armor will no longer reduce the Rage you gain" (30 September deep
// dive): the same 1,000-point swing pays the same rage whether the tank's
// armor let 450 or 650 of it through, Kalgan's 2.5 x damage / c on the
// swing before armor at the level's assumed 40% at 60.
func TestRageFromDamageTakenIgnoresTheReceiversArmor(t *testing.T) {
	tank := &Unit{Level: 60}
	boss := &Unit{Level: 63}
	swing := &Spell{SpellSchool: SpellSchoolPhysical}
	want := 1000 * (1 - 0.40) * 2.5 / GetRageConversion(60)
	for _, armorMultiplier := range []float64{0.45, 0.65, 1} {
		result := &SpellResult{Damage: 1000 * armorMultiplier, ResistanceMultiplier: armorMultiplier}
		if got := damageTakenRage(tank, boss, swing, result); math.Abs(got-want) > 1e-9 {
			t.Errorf("rage from a 1,000 swing behind a %v armor multiplier = %v, want %v", armorMultiplier, got, want)
		}
	}
}

// A spell's partial resist is not armor: the damage dealt stays the rage's
// base, at the level's assumed armor applied to nothing.
func TestRageFromSpellDamageTakenKeepsThePartialResist(t *testing.T) {
	tank := &Unit{Level: 60}
	caster := &Unit{Level: 63}
	bolt := &Spell{SpellSchool: SpellSchoolFrost}
	result := &SpellResult{Damage: 750, ResistanceMultiplier: 0.75}
	if got, want := damageTakenRage(tank, caster, bolt, result), 750*2.5/GetRageConversion(60); math.Abs(got-want) > 1e-9 {
		t.Errorf("rage from a partially resisted 1,000 bolt = %v, want %v", got, want)
	}
}
