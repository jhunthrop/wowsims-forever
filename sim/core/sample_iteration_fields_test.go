package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
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
