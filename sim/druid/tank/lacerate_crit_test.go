package tank

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// What the client says about Lacerate's crits (build 1.60.1.70009, spells
// 414644, 1235826, 1235827 and the hidden companion 414647): the direct
// hit is a weapon-percent-damage effect on a spell with no cannot-crit
// attribute, so it crits like any special attack. The bleed is a periodic
// damage aura carrying exactly the attribute set of Rake, Rip and Rend,
// none of which the engine lets crit, and no beta measurement or patch
// note makes any druid bleed an exception (the engine's rule: a dot
// ticks flat until it has been measured critting, core.Dot.CanCrit). This
// pins both halves so a change to either is a decision.
func TestLacerateHitCritsAndTheBleedDoesNot(t *testing.T) {
	bear, sim, target := newBearSim(t, 60, nil)
	bear.AddStatDynamic(sim, stats.Crit, 100*core.CritRatingPerCritChance)

	bear.AddRage(sim, 100, bear.NewRageMetrics(core.ActionID{SpellID: 1}))
	bear.Lacerate.Cast(sim, target)

	deadline := sim.CurrentTime + 20*time.Second
	for sim.CurrentTime < deadline {
		if sim.Step() {
			break
		}
	}

	metrics := bear.Lacerate.SpellMetrics[target.UnitIndex]
	if metrics.Casts == 0 {
		t.Skip("the Lacerate cast did not go out on this seed")
	}
	if metrics.Misses > 0 {
		t.Skip("the Lacerate cast missed on this seed")
	}
	if metrics.Crits == 0 {
		t.Errorf("the direct hit never crit at 100%% crit chance: %+v", metrics)
	}
	if metrics.CritTicks != 0 {
		t.Errorf("the bleed crit %d times; a druid bleed ticks flat until it is measured critting", metrics.CritTicks)
	}
	if metrics.Ticks == 0 {
		t.Error("the bleed never ticked in 20 seconds")
	}
}
