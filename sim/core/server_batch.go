package core

import "time"

// The server does not run events in continuous time: it gathers what
// happens into short update batches and, inside a batch, updates players
// (their casts, channels and aura ticks) before creatures (cmangos-classic
// Map::Update). Since Classic 1.13.7 the batch is 10 ms (Blizzard cut it
// from 400 ms; "usually 1 to 10 ms"). So a hit that lands a hair before a
// player's cast ends or channel ticks, inside that batch, is processed
// after the cast or tick, exactly as a hit at the same instant is.
const serverBatchWindow = 10 * time.Millisecond

const serverBatchLabel = "Server Batch"

// sameServerBatch reports whether an event gap after now still falls in
// the batch now is processed in. The batch grid's phase is arbitrary, so
// a gap shares the batch with a chance that falls from 1 at no gap to 0
// at a whole batch. No gap at all always shares it and draws nothing, so
// exact ties keep the server's order without touching the RNG.
func (sim *Simulation) sameServerBatch(gap time.Duration) bool {
	switch {
	case gap <= 0:
		return true
	case gap >= serverBatchWindow:
		return false
	default:
		return sim.Proc(1-float64(gap)/float64(serverBatchWindow), serverBatchLabel)
	}
}
