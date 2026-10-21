package dpswarrior

import "testing"

// Improved Charge grants its rage once per sim, at Reset, standing in
// for the opening Charge every melee profile uses at the pull
// (talents.go's applyImprovedCharge). This is the arithmetic end to
// end: CurrentRage() once the sim's time-0 events drain, under each
// rank and under none.
func TestImprovedChargeGrantsRageAtThePull(t *testing.T) {
	// baselineRage is read off the no-talent case rather than pinned, so
	// this test does not also have to track whatever StartingRage the
	// shared Fury test options carry.
	var baselineRage float64

	for _, tc := range []struct {
		name  string
		rank  int
		bonus float64
	}{
		{"no talent", 0, 0},
		{"rank 1", 1, 3},
		{"rank 2", 2, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			talents := emptyWarriorTalents
			if tc.rank > 0 {
				talents = talentStringWithRank(t, emptyWarriorTalents, "improved_charge", tc.rank)
			}

			war, sim := buildWarriorAndSimForCostTest(t, talents)

			// The rage grant is deferred one event past Reset (see
			// applyImprovedCharge's comment on why), so draining every
			// pending action still queued at time 0 is what lets it
			// fire before CurrentRage is read.
			for i := 0; i < 50 && sim.CurrentTime == 0; i++ {
				if sim.Step() {
					break
				}
			}

			if tc.rank == 0 {
				baselineRage = war.CurrentRage()
			}
			if got, want := war.CurrentRage(), baselineRage+tc.bonus; got != want {
				t.Errorf("CurrentRage() once time-0 events drain = %v, want %v (baseline %v + %v)", got, want, baselineRage, tc.bonus)
			}
		})
	}
}
