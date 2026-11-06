package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Round 25 (platform): round 22 made an unknown aura read inactive, but a
// condition on a spell the character has not registered (one it learns later,
// a talent it did not take) still built to nil, and and/or/not/compare drop a
// nil operand. Beast Mastery's Serpent Sting and Arcane Shot read "Multi-Shot's
// time to ready >= 0.5 OR mana <= 25%": below 18, with no Multi-Shot, the first
// term vanished and both shots waited for a quarter mana that never came. An
// unknown spell now reads as one the character cannot cast: not ready, not
// castable, ready only after any fight is over, and 0 for its counts and times.
func TestAnUnknownSpellReadsAsOneYouCannotCastInsteadOfDroppingTheGate(t *testing.T) {
	sim := &Simulation{}
	rot := &APLRotation{unit: newBareTestUnit()}
	unknown := &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: 987654}}

	bools := map[string]APLValue{
		"spellCanCast":      rot.newValueSpellCanCast(&proto.APLValueSpellCanCast{SpellId: unknown}),
		"spellIsReady":      rot.newValueSpellIsReady(&proto.APLValueSpellIsReady{SpellId: unknown}),
		"spellInFlight":     rot.newValueSpellInFlight(&proto.APLValueSpellInFlight{SpellId: unknown}),
		"spellIsChanneling": rot.newValueSpellIsChanneling(&proto.APLValueSpellIsChanneling{SpellId: unknown}),
	}
	for name, v := range bools {
		if v == nil || v.GetBool(sim) {
			t.Errorf("%s(unknown) = %v, want a value reading false", name, v)
		}
	}
	durations := map[string]APLValue{
		"spellCastTime":   rot.newValueSpellCastTime(&proto.APLValueSpellCastTime{SpellId: unknown}),
		"spellTravelTime": rot.newValueSpellTravelTime(&proto.APLValueSpellTravelTime{SpellId: unknown}),
	}
	for name, v := range durations {
		if v == nil || v.GetDuration(sim) != 0 {
			t.Errorf("%s(unknown) = %v, want a value reading 0", name, v)
		}
	}
	for name, v := range map[string]APLValue{
		"spellCpm":            rot.newValueSpellCPM(&proto.APLValueSpellCPM{SpellId: unknown}),
		"spellChanneledTicks": rot.newValueSpellChanneledTicks(&proto.APLValueSpellChanneledTicks{SpellId: unknown}),
		"spellCurrentCost":    rot.newValueSpellCurrentCost(&proto.APLValueSpellCurrentCost{SpellId: unknown}),
	} {
		if v == nil || v.GetFloat(sim) != 0 {
			t.Errorf("%s(unknown) = %v, want a value reading 0", name, v)
		}
	}

	ttr := rot.newValueSpellTimeToReady(&proto.APLValueSpellTimeToReady{SpellId: unknown})
	if ttr == nil {
		t.Fatal("spellTimeToReady(unknown) built to nil: the gate drops")
	}
	if got := ttr.GetDuration(sim); got < unknownSpellTimeToReady {
		t.Fatalf("spellTimeToReady(unknown) = %v, want at least %v", got, unknownSpellTimeToReady)
	}
	// The Beast Mastery gate: "Multi-Shot's time to ready >= 0.5" reads true
	// with no Multi-Shot (nothing shares the cooldown), so the shot is fired.
	gate := &APLValueCompare{op: proto.APLValueCompare_OpGe, lhs: ttr,
		rhs: rot.newValueConst(&proto.APLValueConst{Val: "500ms"})}
	if gate == nil || !gate.GetBool(sim) {
		t.Fatal("timeToReady(unknown) >= 0.5 does not read true")
	}
}
