package rogue

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// TestRankIndexClampsAnOverRankToTheTablesMaxEntry mirrors
// mage/talents_test.go's test of the same name: core.FillTalentsProto
// does not validate a talent string against the client's per-node max
// rank, so a corrupt or hand-edited string could otherwise index
// setupComboPointChance out of range instead of reading Setup's
// max-rank (3) value.
func TestRankIndexClampsAnOverRankToTheTablesMaxEntry(t *testing.T) {
	if got, want := setupComboPointChance[rankIndex(9, setupComboPointChance[:])], setupComboPointChance[len(setupComboPointChance)-1]; got != want {
		t.Errorf("setupComboPointChance at rank 9 = %v, want the max-rank value %v", got, want)
	}
	// A rank inside the table still reads its own entry, not the max.
	if got, want := setupComboPointChance[rankIndex(2, setupComboPointChance[:])], setupComboPointChance[2]; got != want {
		t.Errorf("setupComboPointChance at rank 2 = %v, want %v", got, want)
	}
	// A negative rank (shouldn't occur, but rankIndex's own bounds check
	// covers it) must not panic either.
	if got, want := setupComboPointChance[rankIndex(-1, setupComboPointChance[:])], setupComboPointChance[len(setupComboPointChance)-1]; got != want {
		t.Errorf("setupComboPointChance at rank -1 = %v, want the max-rank value %v", got, want)
	}
}

// TestSetupComboPointChanceTableValues pins Setup's 70755 trait curve
// (TraitDefinitionEffectPoints 23403, curve 83009): 15/30/45%.
func TestSetupComboPointChanceTableValues(t *testing.T) {
	want := [4]float64{0, 0.15, 0.30, 0.45}
	if setupComboPointChance != want {
		t.Errorf("setupComboPointChance = %v, want %v", setupComboPointChance, want)
	}
}

// TestInitiativeComboPointChanceTableValues pins the client's own 33%/
// 67%/100% rank text (rogue.json's Initiative node), the same shape as
// Setup above: "Gives you a 33%/67%/100% chance to add an additional combo
// point ... when using your Ambush, Garrote, or Cheap Shot ability." The
// engine used a flat 25% per rank (25/50/75).
func TestInitiativeComboPointChanceTableValues(t *testing.T) {
	want := [4]float64{0, 0.33, 0.67, 1.0}
	if initiativeComboPointChance != want {
		t.Errorf("initiativeComboPointChance = %v, want %v", initiativeComboPointChance, want)
	}
}

// TestSealFateFollowsTheClientCurve pins 14186's 70755 curve (83040):
// 33/66/100% over three ranks.
func TestSealFateFollowsTheClientCurve(t *testing.T) {
	if want := [4]float64{0, 0.33, 0.66, 1.0}; sealFateChance != want {
		t.Errorf("sealFateChance = %v, want %v", sealFateChance, want)
	}
}

// TestRelentlessStrikesFollowsTheClientCurve pins 14179's 70755 curve
// (128113): 4% per combo point per rank, five ranks.
func TestRelentlessStrikesFollowsTheClientCurve(t *testing.T) {
	for rank, want := range map[int32]float64{0: 0, 1: 0.04, 3: 0.12, 5: 0.20, 9: 0.20} {
		if got := relentlessStrikesChancePerComboPoint(rank); got != want {
			t.Errorf("Relentless Strikes rank %d = %v per combo point, want %v", rank, got, want)
		}
	}
}

// TestComposureDeepensPoised pins 1329265's curve (128092): Poised's -10
// becomes -15 / -20.
func TestComposureDeepensPoised(t *testing.T) {
	for rank, want := range map[int32]int64{0: -10, 1: -15, 2: -20} {
		r := &Rogue{Talents: &proto.RogueTalents{Composure: rank}}
		if got := r.poisedCostReduction(); got != want {
			t.Errorf("Poised with Composure %d = %d, want %d", rank, got, want)
		}
	}
}

// TestDeathMarkFollowsItsRankText pins the cap and storage formula: rank 1
// at 5 combo points with 1000 AP and 20% crit is (63 + 5x102 + 5x0.04x1000)
// x 1.2 = 927.6, storing 60% (Garrote and Rupture 120%); Coup de Grace 3
// raises both by 20%.
func TestDeathMarkFollowsItsRankText(t *testing.T) {
	if got, want := deathMarkCap(1, 5, 1000, 20, 0), (63.0+510+200)*1.2; math.Abs(got-want) > 1e-9 {
		t.Errorf("Death Mark rank 1 cap = %v, want %v", got, want)
	}
	if got, want := deathMarkCap(4, 5, 1000, 20, 0.20), (156.0+5*245+200)*1.2*1.2; math.Abs(got-want) > 1e-9 {
		t.Errorf("Death Mark rank 4 cap with Coup de Grace = %v, want %v", got, want)
	}
	if got := deathMarkStorageRate(5, 0, false); math.Abs(got-0.60) > 1e-9 {
		t.Errorf("Death Mark 5-point storage = %v, want 0.60", got)
	}
	if got := deathMarkStorageRate(5, 0, true); math.Abs(got-1.20) > 1e-9 {
		t.Errorf("Death Mark 5-point bleed storage = %v, want 1.20", got)
	}
}
