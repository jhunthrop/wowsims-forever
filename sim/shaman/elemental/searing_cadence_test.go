package elemental

import (
	"testing"
	"time"
)

// searingTotemLoggedCadence is the median time between a Searing Totem's
// successive landed Attacks in foreverlogs.gg beta timelines (round 13
// casters: 1,072 gaps over ranks 1 to 3, medians 2,433 to 2,438 ms). The
// client's Attack is a 2,200 ms cast on every rank (SpellMisc
// CastingTimeIndex 24, 3606 to 10436), and the cadence does not move with
// the bolt's flight (0 to 1.4 s flights all read 2.43 to 2.44 s), so the
// totem does not wait for its bolt to land.
const searingTotemLoggedCadence = 2437 * time.Millisecond

func TestSearingTotemAttacksAtTheLoggedCadence(t *testing.T) {
	sim, s := newFireNovaShaman(t, 60, "")
	dropSearingTotem(sim, s)
	totem := s.SearingTotem[len(s.SearingTotem)-1]
	dot := totem.Dot(sim.GetTargetUnit(0))
	if got := dot.TickLength; got != searingTotemLoggedCadence {
		t.Errorf("Searing Totem attacks every %v, want the logged %v", got, searingTotemLoggedCadence)
	}
	// 55 s of totem at that cadence is 22 attacks.
	if got, want := dot.NumberOfTicks, int32(22); got != want {
		t.Errorf("Searing Totem rank 6 makes %d attacks, want %d", got, want)
	}
}
