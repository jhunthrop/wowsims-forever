package shadow

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/priest"
)

// The client's Shadowform text: "reducing the Mana cost of all Shadow
// spells by 50%, increasing the critical strike damage bonus of your Shadow
// spells by 100%". Mind Blast is a Shadow spell, but it carried no class
// mask, so the ClassSpellsOnly mods skipped it: full mana cost and a plain
// crit bonus while Mind Flay and Shadow Word: Pain got both (audit P3).
func TestShadowformReachesMindBlast(t *testing.T) {
	sim, built, _ := newShadowPriestSimWithTalents(t, map[string]int{"shadowform": 1})
	mb := built.MindBlast[priest.MindBlastRanks]
	if mb == nil {
		t.Fatal("top Mind Blast rank not registered at level 60")
	}
	if built.ShadowformAura.IsActive() {
		built.ShadowformAura.Deactivate(sim)
	}
	baseCost := mb.Cost.GetCurrentCost()
	baseBonus := mb.CritDamageBonus

	built.ShadowformAura.Activate(sim)
	if got, want := mb.Cost.GetCurrentCost(), baseCost/2; math.Abs(got-want) > 0.5 {
		t.Errorf("Mind Blast costs %v in Shadowform, want %v (half of %v)", got, want, baseCost)
	}
	if got, want := mb.CritDamageBonus-baseBonus, 1.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("Shadowform raised Mind Blast's crit damage bonus by %v, want %v", got, want)
	}
}
