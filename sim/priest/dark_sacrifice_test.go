package priest

import (
	"math"
	"testing"
)

// Client rows (SpellEffect, spells 1277324 to 1277328): the aura takes and
// gives 80/136/.../320 per 3 s for 15 s, so rank 1 totals 400 and rank 5
// 1600; the text "${$o2+$SPI}" adds 100% of Spirit to the mana side once.
func TestDarkSacrificeTotalsMatchTheClientRows(t *testing.T) {
	for rank, wantHealth := range map[int]float64{1: 400, 5: 1600} {
		if got := DarkSacrificeBaseDamage[rank][0] * float64(darkSacrificeTicks); got != wantHealth {
			t.Errorf("rank %d health over the cast = %v, want %v", rank, got, wantHealth)
		}
	}
}

func TestDarkSacrificeManaAddsAllOfSpiritOverTheCast(t *testing.T) {
	const spirit = 250.0
	for rank := 1; rank <= DarkSacrificeRanks; rank++ {
		tick := DarkSacrificeBaseDamage[rank][0]
		total := 0.0
		for i := int32(0); i < darkSacrificeTicks; i++ {
			total += darkSacrificeManaPerTick(tick, spirit)
		}
		want := tick*float64(darkSacrificeTicks) + spirit
		if math.Abs(total-want) > 1e-9 {
			t.Errorf("rank %d mana over the cast = %v, want %v", rank, total, want)
		}
	}
}

// SpellEffect's EffectRealPointsPerLevel is 1 on every rank's two effects and
// SpellLevels caps the growth at 28/38/48/58/68: a priest above the rank's
// level takes and gets a little more a tick than the rank's own row, and at
// the rank's level exactly the row.
func TestDarkSacrificeTickGrowsOneALevelAboveItsRank(t *testing.T) {
	for _, tc := range []struct {
		rank, level int
		want        float64
	}{
		{1, 20, 80}, {1, 25, 85}, {1, 28, 88}, {1, 35, 88},
		{4, 50, 258}, {4, 55, 263},
		{5, 60, 320},
	} {
		if got := darkSacrificeTick(tc.rank, tc.level); got != tc.want {
			t.Errorf("rank %d at level %d: tick %v, want %v", tc.rank, tc.level, got, tc.want)
		}
	}
}
