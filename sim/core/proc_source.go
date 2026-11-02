package core

import "time"

// ProcSource is the cast that granted a one-charge buff and when. A charge
// aura's OnCastComplete runs for the granting cast too (after its hit), so
// it skips exactly that cast at that moment and is spent by any other one,
// an instant completing at the same moment included. Comparing the aura's
// remaining time to its full duration skipped that instant as well.
type ProcSource struct {
	Spell *Spell
	At    time.Duration
}

// IsFrom says whether spell, completing now, is the cast that granted the charge.
func (p ProcSource) IsFrom(sim *Simulation, spell *Spell) bool {
	return p.Spell == spell && p.At == sim.CurrentTime
}
