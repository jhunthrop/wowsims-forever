package elemental

import (
	"testing"
	"time"
)

// The published Elemental build, with Elemental Focus (16164).
const elementalFocusBuild = "5510331300001351-055000000000000000-0530020000000000"

// Clearcasting (16246) has one charge (SpellAuraOptions ProcCharges 1) and
// makes the next damage spell free; the client gives Elemental Focus no
// proc cooldown (16164 has no ProcCategoryRecovery). The first damage cast
// spends it, a heal does not.
func TestClearcastingFreesAndIsSpentByTheNextDamageSpell(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, elementalFocusBuild)
	aura := built.ClearcastingAura
	if aura == nil {
		t.Fatal("the build has no Clearcasting")
	}
	bolt := built.LightningBolt[len(built.LightningBolt)-1]
	full := bolt.Cost.GetCurrentCost()
	aura.Activate(sim)
	aura.SetStacks(sim, aura.MaxStacks) // as the Elemental Focus proc does
	if got := bolt.Cost.GetCurrentCost(); got != 0 {
		t.Errorf("a clearcast Lightning Bolt costs %v, want 0 (full %v)", got, full)
	}
	sim.CurrentTime += time.Second
	if n := len(built.HealingWave); n > 0 && built.HealingWave[n-1] != nil {
		aura.OnCastComplete(aura, sim, built.HealingWave[n-1])
		if !aura.IsActive() {
			t.Error("a Healing Wave spent Clearcasting, which only damage spells use")
		}
	}
	aura.OnCastComplete(aura, sim, bolt)
	if aura.IsActive() {
		t.Error("Lightning Bolt did not spend Clearcasting")
	}
	if got := bolt.Cost.GetCurrentCost(); got != full {
		t.Errorf("after the charge, Lightning Bolt costs %v, want %v", got, full)
	}
}
