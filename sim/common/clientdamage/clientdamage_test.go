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
