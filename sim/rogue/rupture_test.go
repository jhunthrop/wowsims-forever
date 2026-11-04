package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
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
	wantPerComboPoint := [7]float64{0, 1.18, 1.78, 2.37, 2.96, 4.14, 4.73}
	if ruptureComboTickDamage != wantPerComboPoint {
		t.Errorf("ruptureComboTickDamage = %v, want %v", ruptureComboTickDamage, wantPerComboPoint)
	}
}

// TestRuptureTicksPerComboPoint pins the client's 8/10/12/14/16 s durations
// for 1-5 combo points at one tick per 2 s.
func TestRuptureTicksPerComboPoint(t *testing.T) {
	r := &Rogue{Talents: &proto.RogueTalents{}}
	for comboPoints, wantSeconds := range map[int32]int{1: 8, 2: 10, 3: 12, 4: 14, 5: 16} {
		if got := int(r.RuptureDuration(comboPoints).Seconds()); got != wantSeconds {
			t.Errorf("Rupture duration at %d combo points = %ds, want %ds", comboPoints, got, wantSeconds)
		}
	}
}

// TestBloodSplatterLengthensRuptureAndGarrote pins 1329256's 70755 curves:
// Rupture +6 sec (three 2 s ticks), Garrote +9 sec (three 3 s ticks).
func TestBloodSplatterLengthensRuptureAndGarrote(t *testing.T) {
	r := &Rogue{Talents: &proto.RogueTalents{BloodSplatter: true}}
	if got := int(r.RuptureDuration(5).Seconds()); got != 22 {
		t.Errorf("Rupture at 5 combo points with Blood Splatter = %ds, want 22s", got)
	}
	if got := r.bloodSplatterTicks(bloodSplatterGarroteExtra, garroteTickLength); got != 3 {
		t.Errorf("Blood Splatter Garrote ticks = %d, want 3 (9 sec)", got)
	}
}
