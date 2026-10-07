package clientdamage

import (
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

func TestRollGrowsWithLevelUpToTheCapAndKeepsTheWidth(t *testing.T) {
	own := []float64{100, 120} // centre 110
	cases := []struct {
		name                            string
		ppl                             float64
		spellLevel, maxLevel, casterLvl int
		want                            [2]float64
	}{
		{"below the spell's level", 2, 20, 0, 10, [2]float64{100, 120}},
		{"at the spell's level", 2, 20, 0, 20, [2]float64{100, 120}},
		{"above it", 2.2, 20, 0, 30, [2]float64{100 * 132 / 110.0, 120 * 132 / 110.0}},
		{"capped", 2.2, 20, 25, 60, [2]float64{100 * 121 / 110, 120 * 121 / 110}},
		{"flat roll", 0, 20, 0, 60, [2]float64{100, 120}},
	}
	for _, tc := range cases {
		got := Roll(own, tc.ppl, tc.spellLevel, tc.maxLevel, tc.casterLvl)
		if abs(got[0]-tc.want[0]) > 1e-9 || abs(got[1]-tc.want[1]) > 1e-9 {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestRollAgreesWithTheClientTable(t *testing.T) {
	client, err := spellconst.Load("../../core/testdata/conformance/client/mage.json")
	if err != nil {
		t.Fatal(err)
	}
	spell, _ := client.ByID(25306) // Fireball 12
	for _, level := range []int{10, 38, 60, 70} {
		low, high, _ := spell.DamageRange(0, spell.SpellLevel)
		effect := spell.Effects[0]
		got := Roll([]float64{low, high}, effect.PointsPerLevel, spell.SpellLevel, spell.MaxLevel, level)
		AssertRoll(t, client, "Fireball 12", 25306, 0, level, got)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
