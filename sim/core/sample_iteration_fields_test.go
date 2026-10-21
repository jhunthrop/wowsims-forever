package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestSampleIterationFieldsExist(t *testing.T) {
	result := &proto.RaidSimResult{
		SampleIteration: &proto.SampleIteration{
			Dps:             1204.5,
			DurationSeconds: 180,
			Casts: []*proto.SampleCast{
				{
					AtMs:      -1500,
					ActionId:  &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: 11565}},
					Target:    "Target 1",
					Resources: map[string]int32{"rage": 20},
				},
			},
		},
	}

	sample := result.GetSampleIteration()
	if sample.GetDps() != 1204.5 {
		t.Errorf("Dps = %v, want 1204.5", sample.GetDps())
	}
	if len(sample.GetCasts()) != 1 {
		t.Fatalf("Casts = %v, want one", sample.GetCasts())
	}
	cast := sample.GetCasts()[0]
	if cast.GetAtMs() != -1500 {
		t.Errorf("AtMs = %d, want -1500 so a pre-pull cast reads as negative", cast.GetAtMs())
	}
	if cast.GetActionId().GetSpellId() != 11565 {
		t.Errorf("ActionId.SpellId = %d, want 11565", cast.GetActionId().GetSpellId())
	}
	if cast.GetResources()["rage"] != 20 {
		t.Errorf("Resources[rage] = %d, want 20", cast.GetResources()["rage"])
	}
}

func TestSimOptionsHasTheSampleFlag(t *testing.T) {
	if !(&proto.SimOptions{SampleIteration: true}).GetSampleIteration() {
		t.Error("SimOptions.SampleIteration did not round-trip")
	}
}

// The two field numbers themselves, read off the descriptor rather than
// off the .proto text. They are a wire contract with the site - the
// numbers are pinned in
// docs/superpowers/specs/2026-09-19-simulator-parity-interfaces.md
// section 5 - and the getters above go on compiling through a
// renumbering, so this is the only thing that notices one. Same shape as
// TestEncounterParityFieldNumbers.
func TestSampleIterationFieldNumbers(t *testing.T) {
	for _, tc := range []struct {
		message protoreflect.ProtoMessage
		field   string
		want    int32
	}{
		{&proto.RaidSimResult{}, "sample_iteration", 8},
		{&proto.SimOptions{}, "sample_iteration", 10},
	} {
		descriptor := tc.message.ProtoReflect().Descriptor()
		field := descriptor.Fields().ByName(protoreflect.Name(tc.field))
		if field == nil {
			t.Fatalf("%s has no field %q", descriptor.Name(), tc.field)
		}
		if got := int32(field.Number()); got != tc.want {
			t.Errorf("%s.%s is field %d, want %d", descriptor.Name(), tc.field, got, tc.want)
		}
	}
}
