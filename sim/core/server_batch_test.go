package core

import (
	"testing"
	"time"
)

// hitAt runs the fake sim's event loop until at, lands one damaging hit on
// the caster there (PushBackCast, as a raid pulse does), then runs on to
// the end of the fight.
func hitAt(sim *Simulation, unit *Unit, at time.Duration) {
	StartDelayedAction(sim, DelayedActionOptions{
		DoAt:     at,
		OnAction: func(sim *Simulation) { unit.PushBackCast(sim) },
	})
	for sim.CurrentTime < 30*time.Second {
		if sim.Step() {
			return
		}
	}
}

// channelFinalTickSim starts the fake agent's dot (six ticks, one every
// 3 s, the last at 18 s) as the caster's channel.
func channelFinalTickSim(t *testing.T) (*Simulation, *FakeAgent) {
	t.Helper()
	sim := SetupFakeSim()
	fa := sim.Raid.Parties[0].Players[0].(*FakeAgent)
	fa.Dot.Apply(sim)
	fa.Unit.ChanneledDot = fa.Dot
	fa.Unit.channelPushback = 1.0
	if fa.Dot.ExpiresAt() != 18*time.Second {
		t.Fatalf("the fake channel ends at %v, want 18s", fa.Dot.ExpiresAt())
	}
	return sim, fa
}

// A hit that lands in the same server batch as the channel's last tick
// does not take that tick. The server runs a player's spells and auras
// before the creatures in each update (cmangos-classic Map::Update) and a
// periodic aura ticks when its timer reaches zero in that update
// (SpellAuras.cpp Aura::Update), so a hit a hair before the tick, inside
// the 10 ms batch, lands after it. A hit a whole batch earlier still cuts
// the tick.
func TestHitInTheSameBatchAsTheLastChannelTickKeepsIt(t *testing.T) {
	cases := []struct {
		name      string
		before    time.Duration
		wantTicks int32
	}{
		{"1 ns before the last tick", time.Nanosecond, 6},
		{"a quarter millisecond before", 250 * time.Microsecond, 6},
		{"20 ms before, a batch earlier", 20 * time.Millisecond, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sim, fa := channelFinalTickSim(t)
			hitAt(sim, &fa.Unit, 18*time.Second-c.before)
			if fa.Dot.TickCount != c.wantTicks {
				t.Errorf("the channel ticked %d times, want %d", fa.Dot.TickCount, c.wantTicks)
			}
		})
	}
}

// A hit in the same server batch as the end of a cast does not push the
// cast back: the cast completes in the player's update first. A hit a
// batch or more before the end delays it as usual.
func TestHitInTheSameBatchAsTheCastEndDoesNotPushItBack(t *testing.T) {
	cases := []struct {
		name      string
		before    time.Duration
		wantDelay time.Duration
	}{
		{"1 ns before the cast ends", time.Nanosecond, 0},
		{"a quarter millisecond before", 250 * time.Microsecond, 0},
		{"20 ms before, a batch earlier", 20 * time.Millisecond, time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sim := SetupFakeSim()
			fa := sim.Raid.Parties[0].Players[0].(*FakeAgent)
			unit := &fa.Unit
			end := sim.CurrentTime + c.before
			unit.Hardcast = Hardcast{
				Expires:  end,
				ActionID: fa.Spell.ActionID,
				Target:   unit.CurrentTarget,
				Pushback: 1.0,
				CastTime: 2500 * time.Millisecond,
			}
			unit.PushBackCast(sim)
			if got := unit.Hardcast.Expires - end; got != c.wantDelay {
				t.Errorf("the cast was pushed back %v, want %v", got, c.wantDelay)
			}
		})
	}
}

// Two events further apart than a batch are never merged, and a gap inside
// the batch is merged with a chance that falls from 1 at no gap to 0 at a
// whole batch (the batch boundary falls anywhere between them).
func TestSameServerBatchChance(t *testing.T) {
	sim := SetupFakeSim()
	if !sim.sameServerBatch(0) {
		t.Error("two events at the same instant are not in one batch")
	}
	if sim.sameServerBatch(serverBatchWindow) {
		t.Error("two events a whole batch apart share a batch")
	}
	const trials = 4000
	merged := 0
	for i := 0; i < trials; i++ {
		if sim.sameServerBatch(serverBatchWindow / 4) {
			merged++
		}
	}
	if frac := float64(merged) / trials; frac < 0.70 || frac > 0.80 {
		t.Errorf("a quarter-batch gap shared a batch %.2f of the time, want about 0.75", frac)
	}
}
