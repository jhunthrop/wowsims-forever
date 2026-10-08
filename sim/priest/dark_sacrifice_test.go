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
