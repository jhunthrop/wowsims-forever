package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// TestAspectOfTheBeastGivesMeleeAttackPowerAndDeadlyAspectsMelee: 1299447
// (level 60) is aura 99 at 110 melee attack power, 110 mana (Wowhead Forever:
// "increasing Melee Attack Power by 110"), and Deadly Aspects' text procs its
// melee half "While Aspect of the Beast is active". The engine registered no
// Aspect of the Beast.
func TestAspectOfTheBeastGivesMeleeAttackPowerAndDeadlyAspectsMelee(t *testing.T) {
	// The adopted Survival raid build: Deadly Aspects 5/5.
	sim, hunter := buildLevel60Hunter(t, "5000000000000000-3050051010000000-500230231051022131")
	if hunter.AspectOfTheBeast == nil {
		t.Fatal("a level-60 hunter has no Aspect of the Beast")
	}
	if got := hunter.AspectOfTheBeast.ActionID.SpellID; got != 1299447 {
		t.Errorf("Aspect of the Beast is spell %d at 60, want 1299447", got)
	}
	before := hunter.GetStat(stats.AttackPower)
	hunter.AspectOfTheBeast.ApplyEffects(sim, hunter.CurrentTarget, hunter.AspectOfTheBeast)
	if got := hunter.GetStat(stats.AttackPower) - before; math.Abs(got-110) > 1e-9 {
		t.Errorf("Aspect of the Beast adds %v melee attack power, want 110", got)
	}

	da := hunter.GetAura("Deadly Aspects")
	quick := hunter.GetAura("Quick Strikes")
	if da == nil || quick == nil {
		t.Fatal("Deadly Aspects 5/5 registers no proc")
	}
	hit := &core.SpellResult{Outcome: core.OutcomeHit, Target: hunter.CurrentTarget}
	for i := 0; i < 500 && !quick.IsActive(); i++ {
		da.OnSpellHitDealt(da, sim, hunter.AutoAttacks.MHAuto(), hit)
	}
	if !quick.IsActive() {
		t.Error("Deadly Aspects never procced Quick Strikes from a white swing under Aspect of the Beast")
	}
}
