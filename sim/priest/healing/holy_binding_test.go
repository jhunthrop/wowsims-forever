package healing

import (
	"math"
	"testing"
)

// holyBinding is the Holy tree's Holy Binding, 1-based in tier-then-column
// order (tier 3, column 3).
const holyBinding = 12

// selfHealFromFlashHeal casts the top Flash Heal on target with no crit and
// returns what it healed target for and how much health the priest gained,
// starting the priest 3000 below full so the self-heal is never clipped.
func selfHealFromFlashHeal(t *testing.T, talents string, onSelf bool) (healed, selfGained float64) {
	t.Helper()
	sim, agent := healerSim(t, 60, talents)
	priest := &agent.Unit
	priest.RemoveHealth(sim, 3000)
	target := agent.GetMainTarget()
	if onSelf {
		target = priest
	}
	ranks := registered(agent.FlashHeal)
	top := ranks[len(agent.FlashHeal)-1]
	before := priest.CurrentHealth()
	healed = healedBy(sim, top, target)
	if onSelf {
		return healed, 0
	}
	return healed, priest.CurrentHealth() - before
}

// Holy Binding (1328569), text since the 30 October 2026 hotfix: "Each time
// you heal a target other than yourself with Flash Heal, you are healed for
// 50% as much. This healing generates no threat."
func TestHolyBindingHealsThePriestForHalfAFlashHealOnAnother(t *testing.T) {
	healed, gained := selfHealFromFlashHeal(t, onePoint(holyTree, holyBinding, 1), false)
	if healed <= 0 {
		t.Fatal("Flash Heal healed the tank for nothing")
	}
	if math.Abs(gained-healed*0.5) > 1e-6 {
		t.Errorf("Holy Binding healed the priest for %.2f after a %.2f Flash Heal, want half, %.2f", gained, healed, healed*0.5)
	}
	if _, without := selfHealFromFlashHeal(t, "", false); without != 0 {
		t.Errorf("without Holy Binding the priest gained %.2f health from a Flash Heal on the tank, want 0", without)
	}
}

func TestHolyBindingIgnoresAFlashHealOnThePriest(t *testing.T) {
	sim, agent := healerSim(t, 60, onePoint(holyTree, holyBinding, 1))
	priest := &agent.Unit
	priest.RemoveHealth(sim, 3000)
	ranks := registered(agent.FlashHeal)
	top := ranks[len(agent.FlashHeal)-1]
	before := priest.CurrentHealth()
	healed := healedBy(sim, top, priest)
	if got := priest.CurrentHealth() - before; math.Abs(got-healed) > 1e-6 {
		t.Errorf("a self Flash Heal raised the priest by %.2f for a %.2f heal, want exactly the heal (no Holy Binding on self)", got, healed)
	}
}
