package shadow

import "testing"

// Blizzard's 16 October 2026 notes: "Changed Vampiric Embrace so that it
// can't be resisted." The 70790 client carries it as SpellMisc 327025's
// Attributes_3 bit 0x40000 (ignore hit result), which Era's row lacks. Every
// cast on a level-63 boss lands, with no spell hit on the priest.
func TestVampiricEmbraceCannotMiss(t *testing.T) {
	sim, built, target := newShadowPriestSimWithTalents(t, map[string]int{"vampiric_embrace": 1})
	if built.VampiricEmbrace == nil {
		t.Fatal("Vampiric Embrace is not registered with the talent")
	}
	aura := built.VampiricEmbraceAuras.Get(target)
	for i := 0; i < 400; i++ {
		aura.Deactivate(sim)
		built.VampiricEmbrace.CD.Reset()
		built.VampiricEmbrace.ApplyEffects(sim, target, built.VampiricEmbrace)
		if !aura.IsActive() {
			t.Fatalf("Vampiric Embrace missed on cast %d; the client says it cannot", i+1)
		}
	}
}
