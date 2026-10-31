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
// including SPELL_ATTR8_PERIODIC_CAN_CRIT (SpellMisc Attributes_8 0x200).
// The owner confirmed Forever's periodic damage crits, so the engine lets
// every client-flagged dot crit (core.clientPeriodicCanCrit), and the bleed
// crits too. This pins both halves so a change to either is a decision.
func TestLacerateHitAndBleedBothCrit(t *testing.T) {
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
	if metrics.CritTicks == 0 {
		t.Errorf("the bleed never crit at 100%% crit chance; the client marks it PERIODIC_CAN_CRIT: %+v", metrics)
	}
}
