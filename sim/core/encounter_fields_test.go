package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// The three encounter fields the parity design adds are a wire contract
// with the site: the field numbers are pinned in
// docs/superpowers/specs/2026-09-19-simulator-parity-interfaces.md section 5,
// and a renumbering would silently misread every saved request. This test
// is cheap and it is the only thing that notices.
func TestEncounterCarriesTheParityFields(t *testing.T) {
	enc := &proto.Encounter{
		Duration: 180,
		Movement: &proto.MovementPattern{
			IntervalSeconds: 45,
			DurationSeconds: 5,
			CastingOnly:     true,
		},
		TargetsOverTime: []*proto.TargetCountAt{
			{AtSeconds: 0, Count: 1},
			{AtSeconds: 40, Count: 3},
		},
		TargetDummy: true,
	}

	if got := enc.Movement.GetIntervalSeconds(); got != 45 {
		t.Errorf("Movement.IntervalSeconds = %v, want 45", got)
	}
	if got := enc.Movement.GetDurationSeconds(); got != 5 {
		t.Errorf("Movement.DurationSeconds = %v, want 5", got)
	}
	if !enc.Movement.GetCastingOnly() {
		t.Error("Movement.CastingOnly = false, want true")
	}
	if len(enc.TargetsOverTime) != 2 || enc.TargetsOverTime[1].GetCount() != 3 {
		t.Errorf("TargetsOverTime = %v, want the two entries set above", enc.TargetsOverTime)
	}
	if !enc.GetTargetDummy() {
		t.Error("TargetDummy = false, want true")
	}
}

// The field numbers themselves, read off the descriptor rather than off
// the .proto text, so a renumbering fails here.
func TestEncounterParityFieldNumbers(t *testing.T) {
	fields := (&proto.Encounter{}).ProtoReflect().Descriptor().Fields()
	for name, want := range map[string]int32{
		"movement":          10,
		"targets_over_time": 11,
		"target_dummy":      12,
	} {
		field := fields.ByName(protoreflect.Name(name))
		if field == nil {
			t.Fatalf("Encounter has no field %q", name)
		}
		if got := int32(field.Number()); got != want {
			t.Errorf("Encounter.%s is field %d, want %d", name, got, want)
		}
	}
}
