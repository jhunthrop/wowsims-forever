package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestRuptureRankAtLevel(t *testing.T) {
	// Level 60 must keep the rank the old SoD-bracket map gave (6, id 11275).
	if got := core.HighestRankAtLevel(ruptureLearnLevels, 60); got != 6 {
		t.Errorf("rank at 60 = %d, want 6", got)
	}
	// Level 38 is rank 3's own learn level (id 8640) -- matches what the old
	// bracket-40 entry gave, but now also correct for every level in between.
	if got := core.HighestRankAtLevel(ruptureLearnLevels, 38); got != 3 {
		t.Errorf("rank at 38 = %d, want 3", got)
	}
	if got := core.HighestRankAtLevel(ruptureLearnLevels, 19); got != 0 {
		t.Errorf("rank at 19 = %d, want 0", got)
	}
}

// TestRuptureTickDamageByRank pins the per-tick base and per-combo-point
// terms against the client's rank text "(m1 + b1*cp) * ticks" (spell.csv
// description for ids 1943..11275; SpellEffect.csv EffectBasePointsF = m1,
// EffectPointsPerResource = b1, 1.60.1.70009). Before this test every rank
// carried a hand-tuned number: rank 6 was 60 + 8/cp where the client has
// 35 + 4.73/cp.
func TestRuptureTickDamageByRank(t *testing.T) {
	wantBase := [7]float64{0, 5, 7, 11, 16, 22, 35}
	wantPerComboPoint := [7]float64{0, 1.18, 1.78, 2.37, 2.96, 4.14, 4.73}
	if ruptureBaseTickDamage != wantBase {
		t.Errorf("ruptureBaseTickDamage = %v, want %v", ruptureBaseTickDamage, wantBase)
	}
	if ruptureComboTickDamage != wantPerComboPoint {
		t.Errorf("ruptureComboTickDamage = %v, want %v", ruptureComboTickDamage, wantPerComboPoint)
	}
}

// TestRuptureTicksPerComboPoint pins the client's 8/10/12/14/16 s durations
// for 1-5 combo points at one tick per 2 s.
func TestRuptureTicksPerComboPoint(t *testing.T) {
	r := &Rogue{}
	for comboPoints, wantSeconds := range map[int32]int{1: 8, 2: 10, 3: 12, 4: 14, 5: 16} {
		if got := int(r.RuptureDuration(comboPoints).Seconds()); got != wantSeconds {
			t.Errorf("Rupture duration at %d combo points = %ds, want %ds", comboPoints, got, wantSeconds)
		}
	}
}
