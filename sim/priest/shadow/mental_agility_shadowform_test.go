package shadow

import (
	"math"
	"testing"
)

// Mental Agility's cost modifier (14520 effect 0, aura 108 misc 14) names
// Shadowform by class mask (15473's word 0 bit 31, word 1 0x800), so the
// form's 40% of base mana costs 10% less at 3/3, as Shadow Word: Pain's and
// Devouring Plague's do. The engine's Mental Agility mask left it out.
func TestMentalAgilityCutsShadowformsCost(t *testing.T) {
	_, plain, _ := newShadowPriestSimWithTalents(t, map[string]int{"shadowform": 1})
	_, agile, _ := newShadowPriestSimWithTalents(t, map[string]int{"shadowform": 1, "mental_agility": 3})
	if plain.Shadowform == nil || agile.Shadowform == nil {
		t.Fatal("Shadowform is not registered")
	}
	base := plain.Shadowform.Cost.GetCurrentCost()
	if got, want := agile.Shadowform.Cost.GetCurrentCost(), base*0.9; math.Abs(got-want) > 0.5 {
		t.Errorf("Shadowform costs %v with Mental Agility 3/3, want %v (90%% of %v)", got, want, base)
	}
}
