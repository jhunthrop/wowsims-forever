package shaman

import (
	"math"
	"testing"
)

// Sim-accuracy 2026-11-12 (shamans): the client rows the engine's vanilla
// constants had drifted from.

// Lightning Shield's orbs are the client's own spells (26364, 26365, 26366,
// 26367, 26369, 26370, 26363 in 1.60.1.70790 and 70888): 13, 24, 40, 64, 96,
// 134 and 178 Nature damage at a 0.267 coefficient. The 8 Nov advanced log
// has a rank 3 orb landing for 43 to 45 against 12 spell power; the
// vanilla 51 the engine carried would have been 54.
func TestLightningShieldOrbAmountsAreTheClientRows(t *testing.T) {
	want := [LightningShieldRanks + 1]float64{0, 13, 24, 40, 64, 96, 134, 178}
	for rank := 1; rank <= LightningShieldRanks; rank++ {
		if got := LightningShieldBaseDamage[rank]; got != want[rank] {
			t.Errorf("Lightning Shield rank %d orb = %v, want the client's %v", rank, got, want[rank])
		}
	}
}

// Flametongue Weapon's damage is its proc dummy's amount (8026, 8028, 8029,
// 10445, 16343, 16344: EffectBasePointsF plus EffectRealPointsPerLevel to
// SpellLevels.MaxLevel), spread over the weapon's speed as the tooltip
// does ("$m1/77 to $M1/25", speeds 1.3 to 4.0): a hit adds amount x speed /
// 100. Rank 1 is 326 + 19 a level from 10 to 16, rank 4 1144 + 73 from 36
// to 44, rank 6 2498 + 78 from 56 to 64.
func TestFlametongueWeaponDamageGrowsWithLevel(t *testing.T) {
	cases := []struct {
		rank, level int32
		speed, want float64
	}{
		{1, 10, 3.0, 3.26 * 3.0},
		{1, 16, 3.0, 4.40 * 3.0},
		{1, 20, 3.0, 4.40 * 3.0},
		{2, 18, 2.6, 4.79 * 2.6},
		{3, 26, 3.3, 7.16 * 3.3},
		{3, 34, 3.3, 10.52 * 3.3},
		{4, 44, 3.3, 17.28 * 3.3},
		{5, 54, 3.5, 23.72 * 3.5},
		{6, 56, 3.5, 24.98 * 3.5},
		{6, 60, 3.5, 28.10 * 3.5},
	}
	for _, c := range cases {
		if got := flametongueWeaponHitDamage(c.rank, c.level, c.speed); math.Abs(got-c.want) > 1e-6 {
			t.Errorf("Flametongue Weapon rank %d at level %d on a %.1f weapon adds %.3f, want %.3f", c.rank, c.level, c.speed, got, c.want)
		}
	}
}
