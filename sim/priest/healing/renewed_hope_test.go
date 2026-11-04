package healing

import (
	"testing"
	"time"
)

// TestPenanceShortensWeakenedSoulOncePerCast: Renewed Hope takes 1 s a rank
// off the target's Weakened Soul for a heal it names, and Blizzard counts a
// Penance as one such heal, not three. The community bug tracker's issue #288
// ("Penance procs Renewed Hope talent per cast not per hit") was closed by
// Blizzard on 10 October 2026: "Not A Bug. This is a typo in the deep dive."
// So a full Penance channel on a shielded tank cuts Weakened Soul once, with
// the first pulse; the two channel pulses heal with the crit bonus but take
// nothing more off it.
func TestPenanceShortensWeakenedSoulOncePerCast(t *testing.T) {
	sim, agent := healerSim(t, 60, DiscTalents)
	target := agent.GetMainTarget()
	rank := time.Duration(agent.Talents.RenewedHope)
	if rank == 0 {
		t.Fatal("the Discipline build has no Renewed Hope")
	}
	cut := rank * time.Second

	shield := agent.PowerWordShield[10]
	shield.ApplyEffects(sim, target, shield)
	weakened := agent.WeakenedSouls.Get(target)
	full := weakened.RemainingDuration(sim)

	penance := agent.Penance[4]
	penance.BonusCritRating += noCrit
	start := sim.CurrentTime
	penance.ApplyEffects(sim, target, penance)
	for sim.CurrentTime < start+3*time.Second {
		if sim.Step() {
			break
		}
	}
	if hits := penance.SpellMetrics[target.UnitIndex].Hits; hits != 3 {
		t.Fatalf("Penance landed %d pulses, want 3", hits)
	}
	elapsed := sim.CurrentTime - start
	want := full - elapsed - cut
	if got := weakened.RemainingDuration(sim); got != want {
		t.Errorf("after a full Penance Weakened Soul has %v left, want %v (one %v cut for the cast, not %v for three pulses)",
			got, want, cut, 3*cut)
	}
}
