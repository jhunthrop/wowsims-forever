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
	want := [RipRanks + 1]float64{0, 5.5, 9, 10.5, 15.8, 22.5, 31.9}
	if ripTickPerComboPoint != want {
		t.Errorf("ripTickPerComboPoint = %v, want %v", ripTickPerComboPoint, want)
	}
}

// PounceTickDamage is the "Pounce Bleed" spells' effect 0 (SpellEffect.csv
// 9007, 9824, 9826 in 1.60.1.70790), which the vendored spellconst file does
// not carry: 19, 25 and 31 since the 3 November 2026 notes.
func TestPounceTickDamage(t *testing.T) {
	want := [PounceRanks + 1]float64{0, 19, 25, 31}
	for rank := 1; rank <= PounceRanks; rank++ {
		got := PounceTickDamage[rank]
		if got.Amount != want[rank] || got.SpellLevel != PounceLevel[rank] {
			t.Errorf("rank %d: tick %v at spell level %d, want %v at %d", rank, got.Amount, got.SpellLevel, want[rank], PounceLevel[rank])
		}
	}
}

// Ferocious Bite's per-combo-point step is effect 0's EffectPointsPerResource, which the vendored
// spellconst does not carry: 30 / 45 / 74 / 115 / 160 / 181 on 1.60.1.70755 after the server's
// push 113403 (ids 1328618, 22568, 22827, 22828, 22829, 31018; it was 24 / 36 / 59 / 92 / 128 / 147).
// Rip's step above is the same column on the same push.
func TestFerociousBiteDamagePerComboPointIsThePushs(t *testing.T) {
	want := [FerociousBiteRanks + 1]float64{0, 30, 45, 74, 115, 160, 181}
	if ferociousBiteDamagePerComboPoint != want {
		t.Errorf("ferociousBiteDamagePerComboPoint = %v, want %v", ferociousBiteDamagePerComboPoint, want)
	}
}
