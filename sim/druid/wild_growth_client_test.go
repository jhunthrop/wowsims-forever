package druid

import (
	"math"
	"testing"
)

// Wild Growth's ticks take 0.066 of healing power on every rank in client
// 1.60.1.70888 (SpellEffect 1072065, 1235327 and 1235329,
// EffectBonusCoefficient 0.0659999996), twice the 0.033 the site's
// 1.60.1.70790 tables carry. The hand row holds the live value.
func TestWildGrowthTicksTakeTheLiveClientsCoefficient(t *testing.T) {
	const want = 0.066
	for index, rank := range wildGrowthTable {
		if math.Abs(rank.tick.coefficient-want) > 1e-9 {
			t.Errorf("Wild Growth rank %d (%d): tick coefficient %v, client 1.60.1.70888 states %v", index+1, rank.spellID, rank.tick.coefficient, want)
		}
	}
}
