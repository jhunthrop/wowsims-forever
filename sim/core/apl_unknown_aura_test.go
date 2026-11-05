package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Round 22 (warlock lane, platform): a condition that reads an aura the
// character never registered (a talent it did not take) used to build to nil,
// and and/or/not/compare drop a nil operand, so the gate vanished and the line
// ran with no condition. An unknown aura now reads as inactive, 0 stacks and 0
// time remaining, and an unknown dot as inactive with 0 remaining.
func TestAnUnknownAuraReadsInactiveInsteadOfDroppingTheGate(t *testing.T) {
	sim := &Simulation{}
	rot := &APLRotation{unit: newBareTestUnit()}
	unknown := &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: 987654}}
	active := rot.newValueAuraIsActive(&proto.APLValueAuraIsActive{AuraId: unknown})
	if active == nil || active.GetBool(sim) {
		t.Fatalf("auraIsActive(unknown) = %v, want a value reading false", active)
	}
	reaction := rot.newValueAuraIsActiveWithReactionTime(&proto.APLValueAuraIsActiveWithReactionTime{AuraId: unknown})
	if reaction == nil || reaction.GetBool(sim) {
		t.Fatalf("auraIsActiveWithReactionTime(unknown) = %v, want a value reading false", reaction)
	}
	stacks := rot.newValueAuraNumStacks(&proto.APLValueAuraNumStacks{AuraId: unknown})
	if stacks == nil || stacks.GetInt(sim) != 0 {
		t.Fatalf("auraNumStacks(unknown) = %v, want a value reading 0", stacks)
	}
	remaining := rot.newValueAuraRemainingTime(&proto.APLValueAuraRemainingTime{AuraId: unknown})
	if remaining == nil || remaining.GetDuration(sim) != 0 {
		t.Fatalf("auraRemainingTime(unknown) = %v, want a value reading 0", remaining)
	}
	// The gate the warlock lane hit, "stacks >= 3", now reads false instead of vanishing.
	gate := &APLValueCompare{op: proto.APLValueCompare_OpGe, lhs: stacks,
		rhs: rot.newValueConst(&proto.APLValueConst{Val: "3"})}
	if gate.GetBool(sim) {
		t.Fatal("stacks(unknown) >= 3 reads true")
	}
}
