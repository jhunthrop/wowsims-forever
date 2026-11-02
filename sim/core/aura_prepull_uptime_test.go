package core

import (
	"testing"
	"time"
)

// An aura that is up only before the pull (a prepull buff that fades, or a
// demon's aura when the demon is swapped out in the prepull) has no uptime
// in the fight, and one that comes up in the prepull counts from the pull.
// The engine subtracted the fade time from zero instead, so a Demonology
// warlock's Master Demonologist (Imp) read -25% uptime and the Succubus' one,
// up the whole fight, 66.6%.
func TestAuraUptimeCountsOnlyTheFight(t *testing.T) {
	sim := &Simulation{}
	unit := Unit{Type: PlayerUnit, auraTracker: newAuraTracker()}

	prepullOnly := unit.RegisterAura(Aura{Label: "Prepull Only", ActionID: ActionID{SpellID: 1}, Duration: NeverExpires})
	sim.CurrentTime = -54 * time.Second
	prepullOnly.Activate(sim)
	sim.CurrentTime = -45 * time.Second
	prepullOnly.Deactivate(sim)
	if got := prepullOnly.metrics.Uptime; got != 0 {
		t.Errorf("an aura up from -54 s to -45 s has %v of fight uptime, want 0", got)
	}

	acrossPull := unit.RegisterAura(Aura{Label: "Across Pull", ActionID: ActionID{SpellID: 2}, Duration: NeverExpires})
	sim.CurrentTime = -38 * time.Second
	acrossPull.Activate(sim)
	sim.CurrentTime = 180 * time.Second
	acrossPull.Deactivate(sim)
	if got := acrossPull.metrics.Uptime; got != 180*time.Second {
		t.Errorf("an aura up from -38 s to 180 s has %v of fight uptime, want 3m0s", got)
	}
}
