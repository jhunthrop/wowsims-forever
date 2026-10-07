package clientdamage

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// lightningBolt10 is Lightning Bolt rank 10 as spellconst/shaman.json
// states it: centre 196 at level 56, +1.2 a level, 0.108352 wide, capped
// at level 61.
var lightningBolt10 = Effect{Amount: 196, Variance: 0.108352, PerLevel: 1.2, SpellLevel: 56, MaxLevel: 61}

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestRangeAtTheSpellsOwnLevelIsTheClientRoll(t *testing.T) {
	got := lightningBolt10.Range(56)
	if !near(got[0], 185.38) || !near(got[1], 206.62) {
		t.Fatalf("range at 56 = %v, want 185.38-206.62", got)
	}
}

func TestRangeGrowsPerLevelAboveTheSpell(t *testing.T) {
	got := lightningBolt10.Range(60)
	if !near(got[0], 189.92) || !near(got[1], 211.68) {
		t.Fatalf("range at 60 = %v, want 189.92-211.68", got)
	}
}

func TestGrowthStopsAtTheMaxLevel(t *testing.T) {
	if lightningBolt10.Center(70) != lightningBolt10.Center(61) {
		t.Fatalf("centre keeps growing past the cap: %v vs %v", lightningBolt10.Center(70), lightningBolt10.Center(61))
	}
}

func TestZeroMaxLevelIsUncapped(t *testing.T) {
	uncapped := Effect{Amount: 10, PerLevel: 2, SpellLevel: 10}
	if got := uncapped.Center(60); got != 110 {
		t.Fatalf("uncapped centre at 60 = %v, want 110", got)
	}
}

func TestACasterBelowTheSpellGetsTheOwnLevelAmount(t *testing.T) {
	if got := lightningBolt10.Center(40); got != 196 {
		t.Fatalf("centre below the spell's level = %v, want 196", got)
	}
}

func TestRangeAgreesWithSpellconstDamageRange(t *testing.T) {
	spell := spellconst.Spell{
		SpellLevel: 56,
		MaxLevel:   61,
		Effects:    []spellconst.Effect{{Index: 0, Amount: 196, Variance: 0.108352, PointsPerLevel: 1.2}},
	}
	for _, level := range []int{10, 38, 56, 58, 60, 70} {
		wantMin, wantMax, _ := spell.DamageRange(0, level)
		got := lightningBolt10.Range(level)
		if !near(got[0], wantMin) || !near(got[1], wantMax) {
			t.Errorf("level %d: Range %v, DamageRange %v-%v", level, got, wantMin, wantMax)
		}
	}
}

func TestFromRollRoundTripsTheOwnLevelRoll(t *testing.T) {
	// Wrath rank 8 as constants_auto_gen.go would state it: {91.57, 102.43}
	// at level 54, growing 1.1 a level.
	got := FromRoll([]float64{91.57, 102.43}, 1.1, 54, 61)
	own := got.Range(54)
	if !near(own[0], 91.57) || !near(own[1], 102.43) {
		t.Fatalf("own-level range = %v, want 91.57-102.43", own)
	}
	if !near(got.Center(55), got.Center(54)+1.1) {
		t.Fatalf("centre at 55 = %v, want one level of growth over %v", got.Center(55), got.Center(54))
	}
}

func TestFromRollOfAFlatEffectHasNoWidth(t *testing.T) {
	got := FromRoll([]float64{34, 34}, 0, 54, 0)
	if r := got.Range(60); r[0] != 34 || r[1] != 34 {
		t.Fatalf("flat effect range = %v, want 34-34", r)
	}
}

func TestFromRollRejectsAMalformedRow(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for a roll without two ends")
		}
	}()
	FromRoll([]float64{5}, 0, 1, 0)
}

func TestFromTableBuildsOneEffectPerRank(t *testing.T) {
	got := FromTable([][]float64{{0, 0}, {10, 14}, {20, 20}}, []float64{0, 1, 0}, []int{0, 5, 10}, []int{0, 0, 0})
	if len(got) != 3 {
		t.Fatalf("got %d effects, want 3", len(got))
	}
	if !near(got[1].Center(15), 22) {
		t.Fatalf("rank 1 centre at 15 = %v, want 22", got[1].Center(15))
	}
	if got[2].Variance != 0 {
		t.Fatalf("a flat rank has variance %v", got[2].Variance)
	}
}

func TestFromTableRejectsColumnsOfDifferentLength(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for ragged columns")
		}
	}()
	FromTable([][]float64{{1, 1}}, []float64{0, 0}, []int{0}, []int{0})
}
