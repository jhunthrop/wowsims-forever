package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// The server's hotfix push for 1.60.1.70753 (live on wago.tools
// 2026-11-03 14:14 UTC, SpellEffect rows layered by the site's
// pipeline.hotfix_push) raised the hunter's shots. These pin the pushed
// EffectBasePointsF of every rank, so a table that still carries the
// shipped client's number fails here by name. Wowhead's Forever tooltips
// still showed the shipped text when the push landed.
func TestHotfixPush70753HunterAmounts(t *testing.T) {
	cases := []struct {
		name string
		got  func(rank int) float64
		want []float64 // indexed by rank label, rank 0 unused
	}{
		{"Aimed Shot", func(r int) float64 { return AimedShotDamage[r].Amount },
			[]float64{0, 40, 70, 115, 185, 260, 350}},
		{"Arcane Shot", func(r int) float64 { return ArcaneShotDamage[r].Amount },
			[]float64{0, 25, 35, 50, 85, 120, 175, 220, 280}},
		{"Serpent Sting per tick", func(r int) float64 { return SerpentStingDamage[r].Amount },
			[]float64{0, 3, 7, 14, 26, 40, 58, 75, 100, 135}},
		{"Sniper Shot", func(r int) float64 { return SniperShotDamage[r].Amount },
			[]float64{0, 320, 450, 590}},
		{"Aspect of the Hawk", func(r int) float64 { return AspectOfTheHawkBaseDamage[r][0] },
			[]float64{0, 20, 35, 50, 70, 90, 110, 120}},
	}
	for _, c := range cases {
		for rank := 1; rank < len(c.want); rank++ {
			if got := c.got(rank); got != c.want[rank] {
				t.Errorf("%s rank %d = %v, want the pushed %v", c.name, rank, got, c.want[rank])
			}
		}
	}
}

// Trueshot Aura's five ranks after the push: 20/35/50/75/100 ranged attack
// power (from 15/22/30/40/50).
func TestHotfixPush70753TrueshotAura(t *testing.T) {
	want := map[int32]float64{1299346: 20, 1299348: 35, 19506: 50, 20905: 75, 20906: 100}
	if len(core.TrueshotAuraRanks) != len(want) {
		t.Fatalf("Trueshot Aura has %d ranks, want %d", len(core.TrueshotAuraRanks), len(want))
	}
	for _, rank := range core.TrueshotAuraRanks {
		if rank.Amount != want[rank.SpellID] {
			t.Errorf("Trueshot Aura %d = %v, want the pushed %v", rank.SpellID, rank.Amount, want[rank.SpellID])
		}
	}
}
