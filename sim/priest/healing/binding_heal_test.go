package healing

import "testing"

// Binding Heal (401937, 1240770-1240774; levels 25 to 56): client 1.60.1.70707's
// hotfix push teaches it from the Holy skill line, and Wowhead Forever's
// tooltip for rank 6 (2026-11-01) reads "Heal yourself and a friendly target
// for 778 to 914", 380 mana, a 1.5 sec cast. Both halves land: the target's
// and the priest's.
func TestBindingHealHealsTheTargetAndThePriest(t *testing.T) {
	sim, agent := healerSim(t, 60, "")
	priest := &agent.Unit
	priest.RemoveHealth(sim, 3000)
	ranks := registered(agent.BindingHeal)
	if len(ranks) != 6 {
		t.Fatalf("a level-60 priest has %d Binding Heal ranks, want 6", len(ranks))
	}
	top := ranks[6]
	if top.ActionID.SpellID != 1240774 || top.DefaultCast.Cost != 380 {
		t.Errorf("top rank is %v costing %v, want 1240774 costing 380", top.ActionID, top.DefaultCast.Cost)
	}
	target := agent.GetMainTarget()
	before := priest.CurrentHealth()
	healed := healedBy(sim, top, target)
	self := priest.CurrentHealth() - before
	if healed <= 0 || self <= 0 {
		t.Fatalf("Binding Heal healed the tank for %.1f and the priest for %.1f, want both above zero", healed, self)
	}
	assertWithin(t, "the priest's half", self, healed*0.7, healed*1.3)
}
