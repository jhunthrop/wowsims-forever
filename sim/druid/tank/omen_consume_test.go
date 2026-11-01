package tank

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Clearcasting (16870) has one charge (SpellAuraOptions ProcCharges 1): the
// first eligible ability spends it. Maul is in its class mask (0x800) but
// is cast off the swing with SpellFlagNoOnCastComplete, so the charge must
// be spent by the Maul hit itself, or every Maul for 15 s is free.
func TestMaulSpendsClearcasting(t *testing.T) {
	bear, sim, target := newBearSim(t, 60, nil)
	if bear.ClearcastingAura == nil || bear.Maul == nil {
		t.Fatal("a level-60 bear has no Clearcasting or Maul")
	}
	bear.AddRage(sim, 100, bear.NewRageMetrics(core.ActionID{SpellID: 1}))
	bear.ClearcastingAura.Activate(sim)
	sim.CurrentTime += 1 // past the same-cast guard
	rage := bear.CurrentRage()
	bear.Maul.Cast(sim, target)
	if got := bear.CurrentRage(); got != rage {
		t.Errorf("a clearcast Maul cost %v rage, want none", rage-got)
	}
	if bear.ClearcastingAura.IsActive() {
		t.Error("Maul left Clearcasting up: the next Maul is free too")
	}
}
