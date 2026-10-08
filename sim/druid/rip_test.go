package druid

import (
	"testing"
	"time"
)

// Rip's client tooltip states "damage over $d" for one through five combo
// points alike (spell 9896: duration 12000 ms, 2 s period), so the
// duration is fixed and the combo points only scale the damage per tick.
func TestRipDurationIsFixedAtTwelveSeconds(t *testing.T) {
	if got, want := time.Duration(RipNumberOfTicks)*2*time.Second, 12*time.Second; got != want {
		t.Errorf("Rip lasts %v, want %v (the client's duration_ms 12000 on every rank)", got, want)
	}
}

// ripTickPerComboPoint is the client's EffectPointsPerResource for effect 0
// of each rank: SpellEffect.csv 1079, 9492, 9493, 9752, 9894, 9896.
func TestRipComboPointStepsMatchClient(t *testing.T) {
	want := [RipRanks + 1]float64{0, 4.4, 7.2, 8.5, 12.7, 18.2, 25.5}
	if ripTickPerComboPoint != want {
		t.Errorf("ripTickPerComboPoint = %v, want %v", ripTickPerComboPoint, want)
	}
}

// PounceTickDamage is the "Pounce Bleed" spells' effect 0 (SpellEffect.csv
// 9007, 9824, 9826), which the vendored spellconst file does not carry.
func TestPounceTickDamage(t *testing.T) {
	want := [PounceRanks + 1]float64{0, 15, 20, 25}
	for rank := 1; rank <= PounceRanks; rank++ {
		got := PounceTickDamage[rank]
		if got.Amount != want[rank] || got.SpellLevel != PounceLevel[rank] {
			t.Errorf("rank %d: tick %v at spell level %d, want %v at %d", rank, got.Amount, got.SpellLevel, want[rank], PounceLevel[rank])
		}
	}
}
