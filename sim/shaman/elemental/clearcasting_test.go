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

// Round 9 (casters lane): Clearcasting from Lightning Bolt A must be spent by
// the next damage cast B even when B is an instant that completes at the same
// moment (Fire Nova, a shock). The guard that keeps A from spending its own
// proc compared the aura's remaining time to its full duration, so an
// instant cast at the proc's timestamp was free and left the charge for a
// second free spell. Bare band 50 measured 6.7 free casts a fight from 4.3
// Clearcasting procs.
func TestAnInstantAtTheProcsMomentSpendsClearcasting(t *testing.T) {
	sim, built := newFireNovaShaman(t, 60, elementalTalentsWith(map[int]int{elementalFocusNode: 1}))
	aura := built.ClearcastingAura
	bolt := built.LightningBolt[len(built.LightningBolt)-1]
	nova := topFireNova(t, built)
	built.ProcClearcasting(sim, bolt)
	aura.OnCastComplete(aura, sim, bolt)
	if !aura.IsActive() {
		t.Fatal("the Lightning Bolt that procced Clearcasting spent it")
	}
	aura.OnCastComplete(aura, sim, nova)
	if aura.IsActive() {
		t.Error("a Fire Nova cast at the proc's moment did not spend Clearcasting")
	}
}
