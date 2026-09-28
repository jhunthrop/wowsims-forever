package druid

import (
	"testing"
	"time"
)

// TestRipTicksScalesWithComboPoints pins the fix in this lane: Rip's
// duration must follow combo points instead of the fixed 6 ticks (12 s)
// every rank previously got regardless of how many combo points were
// spent -- the bug that made Ferocious Bite unreachable, since Rip never
// got close to expiring at low combo point counts.
func TestRipTicksScalesWithComboPoints(t *testing.T) {
	druid := &Druid{}

	onePointTicks := druid.RipTicks(1)
	fivePointTicks := druid.RipTicks(5)

	if fivePointTicks <= onePointTicks {
		t.Fatalf("a 5-combo-point Rip (%d ticks) must last longer than a 1-combo-point Rip (%d ticks)", fivePointTicks, onePointTicks)
	}

	// Classic's published table: 8/10/12/14/16 sec for 1-5 combo points,
	// i.e. 3 base ticks @ 2s plus 1 tick per combo point.
	for comboPoints, wantSeconds := range map[int32]int32{1: 8, 2: 10, 3: 12, 4: 14, 5: 16} {
		if got := druid.RipTicks(comboPoints) * 2; got != wantSeconds {
			t.Errorf("Rip duration at %d combo points = %d sec, want %d sec", comboPoints, got, wantSeconds)
		}
	}
}

// TestRipDurationMatchesTicksTimesTickLength pins RipDuration against
// RipTicks directly, so the two helpers can't drift apart.
func TestRipDurationMatchesTicksTimesTickLength(t *testing.T) {
	druid := &Druid{}

	for comboPoints := int32(1); comboPoints <= 5; comboPoints++ {
		want := time.Duration(druid.RipTicks(comboPoints)) * time.Second * 2
		if got := druid.RipDuration(comboPoints); got != want {
			t.Errorf("RipDuration(%d) = %v, want %v", comboPoints, got, want)
		}
	}
}
