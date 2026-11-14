package dpswarrior

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/warrior"
)

// Every Rend tick in our reports runs above the client's flat amount, and the
// gap grows with the warrior's attack power. The one warrior whose talents
// and stance a log names (the 8 November advanced report: COMBATANT_INFO with
// no Improved Rend, Defensive Stance cast before every Rend) ticks 16 to 20
// over 45 applications at a cast-time attack power of 599 to 769, which is
// Defensive Stance's 0.9 times rank 4's 11 plus 0.0129 of attack power a tick
// (bootstrap 0.0126 to 0.0132). A tick snapshots that share at the cast.
func TestRendTickAddsAShareOfAttackPower(t *testing.T) {
	sim, war := newWarriorSim(t, warrior.ForeverFuryTalents)
	if war.Rend == nil {
		t.Fatal("a level 60 warrior should train Rend")
	}
	target := sim.Encounter.TargetUnits[0]
	dot := war.Rend.Dot(target)

	snapshot := func() float64 {
		dot.Apply(sim)
		damage := dot.SnapshotBaseDamage
		dot.Deactivate(sim)
		return damage
	}
	low := snapshot()
	const extraAttackPower = 1000
	war.AddStatDynamic(sim, stats.AttackPower, extraAttackPower)
	high := snapshot()

	if got, want := (high-low)/extraAttackPower, 0.0129; math.Abs(got-want) > 1e-9 {
		t.Errorf("Rend's tick gains %.5f a point of attack power, want %.4f", got, want)
	}
}
