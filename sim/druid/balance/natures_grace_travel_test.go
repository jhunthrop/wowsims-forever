package balance

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// TestNaturesGraceFromAWrathCritStartsWhenTheBoltLands: Nature's Grace
// (16880, ProcTypeMask 0x14000) procs on the spell hit, and Wrath flies
// at 20 yd/s, so a crit's 3 s of haste starts when the bolt lands, not
// when the cast ends. foreverlogs.gg beta timelines (round 13 casters):
// all 31 Wrath crits with a separable flight (182 to 1398 ms) put the
// Nature's Grace buff_applied on the Wrath damage event, none on the cast.
func TestNaturesGraceFromAWrathCritStartsWhenTheBoltLands(t *testing.T) {
	const naturesGraceField = 13

	built, sim, target := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{naturesGraceField: 1}))
	if built.NaturesGraceProcAura == nil {
		t.Fatal("Nature's Grace is not registered")
	}
	wrath := built.Wrath[len(built.Wrath)-1].Spell
	wrath.BonusCritRating += 100 * core.CritRatingPerCritChance
	travel := wrath.TravelTime()
	if travel <= 0 {
		t.Fatalf("Wrath travel time %v from 30 yd, want a flight", travel)
	}

	start := sim.CurrentTime
	wrath.ApplyEffects(sim, target, wrath)
	if built.NaturesGraceProcAura.IsActive() {
		t.Fatalf("Nature's Grace is up at the cast's end (%v), before the Wrath crit has landed", start)
	}

	for sim.CurrentTime < start+travel {
		if sim.Step() {
			t.Fatal("the sim ended before Wrath landed")
		}
	}
	if !built.NaturesGraceProcAura.IsActive() {
		t.Fatalf("Nature's Grace is not up when the Wrath crit lands at %v", sim.CurrentTime)
	}
	if got, want := built.NaturesGraceProcAura.ExpiresAt(), start+travel+built.NaturesGraceProcAura.Duration; got != want {
		t.Errorf("Nature's Grace expires at %v, want the landing plus 3 s (%v)", got, want)
	}
}
