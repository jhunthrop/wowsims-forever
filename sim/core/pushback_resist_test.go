package core

import (
	"testing"
	"time"
)

// A unit's own pushback resist (Concentration Aura) adds to the cast
// spell's (Spiritual Focus) before the one roll: 70% and 35% together
// keep every cast, where either alone loses some.
func TestPushbackResistAddsToTheSpellsOwnChance(t *testing.T) {
	const trials = 2000
	cases := []struct {
		name     string
		spell    float64
		unit     float64
		wantNone bool
	}{
		{"neither", 0, 0, false},
		{"Spiritual Focus alone", 0.7, 0, false},
		{"Concentration Aura alone", 0, 0.35, false},
		{"both", 0.7, 0.35, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sim := SetupFakeSim()
			fa := sim.Raid.Parties[0].Players[0].(*FakeAgent)
			unit := &fa.Unit
			fa.Spell.PushbackReduction = c.spell
			unit.PseudoStats.PushbackResistChance = c.unit
			pushed := 0
			for i := 0; i < trials; i++ {
				end := sim.CurrentTime + 2*time.Second
				unit.Hardcast = Hardcast{
					Expires:  end,
					ActionID: fa.Spell.ActionID,
					Target:   unit.CurrentTarget,
					Pushback: 1.0,
					CastTime: 2500 * time.Millisecond,
				}
				unit.PushBackCast(sim)
				if unit.Hardcast.Expires != end {
					pushed++
				}
			}
			if c.wantNone && pushed != 0 {
				t.Errorf("%d of %d casts were pushed back, want none", pushed, trials)
			}
			if !c.wantNone && pushed == 0 {
				t.Errorf("no cast of %d was pushed back, want some", trials)
			}
		})
	}
}
