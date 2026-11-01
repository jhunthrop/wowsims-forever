package enhancement

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// Stormstrike (17364) effect 1 is aura 271 (damage taken from the
// caster), +20, SpellClassMask_0 0x100003 (Lightning Bolt, Chain
// Lightning, Earth Shock), and its SpellAuraOptions 129538 has ProcCharges
// 1 (1.60.1.70707). The live text: "increase the damage you deal to the
// target with your next Lightning Bolt, Chain Lightning, or Earth Shock
// spell by 20% for 12 sec". So the first of those spells takes the 20% and
// spends it; nothing else of the Nature school is raised.
func TestStormstrikeRaisesOnlyTheNextShock(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{enhNodeStormstrike: 1}, nil))
	target := sim.Encounter.TargetUnits[0]
	earthShock := built.EarthShock[len(built.EarthShock)-1]
	if earthShock == nil {
		t.Fatal("no Earth Shock at 60")
	}
	damage := func() float64 {
		return earthShock.CalcDamage(sim, target, 1000, earthShock.OutcomeAlwaysHit).Damage
	}

	plain := damage()
	natureTakenBefore := target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexNature]
	built.Stormstrike.ApplyEffects(sim, target, built.Stormstrike)

	if got := target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexNature]; math.Abs(got-natureTakenBefore) > 1e-9 {
		t.Errorf("Stormstrike raised all Nature damage taken to %v, want it left at %v", got, natureTakenBefore)
	}
	if got := damage() / plain; math.Abs(got-1.2) > 1e-6 {
		t.Errorf("the next Earth Shock deals %v of its damage, want 1.2", got)
	}
	earthShock.ApplyEffects(sim, target, earthShock)
	if got := damage() / plain; math.Abs(got-1) > 1e-6 {
		t.Errorf("after one Earth Shock the next deals %v of its damage, want 1 (one charge)", got)
	}
}
