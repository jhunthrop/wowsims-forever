package healing

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

// TestInnerFireDoesNotCutIntoThePrepullCasts: the option's Inner Fire is a
// buff a priest puts up long before the pull, as every armor is. Cast at
// -1 s it took a global: the same instant the rotation's Lightwell (cast
// at -2.5 s, 1.5 s) finished, so the priest started Inner Fire while still
// casting Lightwell, which then completed only at +0.5 s, and the first
// heal of the fight waited for that global too.
func TestInnerFireDoesNotCutIntoThePrepullCasts(t *testing.T) {
	for _, c := range []struct {
		file, talents string
	}{
		{"forever_holy", HolyTalents},
		{"forever_discipline", DiscTalents},
	} {
		player := healer(60, c.talents, &proto.HealingPriest_Options{UseInnerFire: true}, loadRotation(t, c.file))
		sim, agent := agentSim(t, player)
		for sim.CurrentTime < 0 && !sim.Step() {
		}
		lightwell := agent.Lightwell[3]
		if lightwell == nil {
			t.Fatalf("%s: no Lightwell rank 3", c.file)
		}
		if got := lightwell.LastCastAt; got != -time.Second {
			t.Errorf("%s: the pre-pull Lightwell completed at %v, want -1s (cast at -2.5s, 1.5s)", c.file, got)
		}
		if aura := agent.GetAura("Inner Fire (Rank 6)"); aura == nil || !aura.IsActive() {
			t.Errorf("%s: Inner Fire is not up at the pull", c.file)
		}
		if ready := agent.GCD.ReadyAt(); ready > 0 {
			t.Errorf("%s: the global cooldown runs to %v at the pull, want ready", c.file, ready)
		}
	}
}
