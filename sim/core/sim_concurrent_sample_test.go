package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// syntheticShardResult builds the minimum *proto.RaidSimResult that
// CombineConcurrentSimResults can walk without panicking: empty Parties and
// Targets (so the per-player/per-target combine loops run zero times), and
// a DistributionMetrics with its own AggregatorData (combineDistMetrics
// dereferences it unconditionally). It exists so this file can test the
// concurrent-combine path's sample-selection wiring without standing up a
// real raid and running real iterations - the combine logic is agnostic to
// what produced its inputs.
func syntheticShardResult(iterationsDone int32, raidDpsAvg float64, sample *proto.SampleIteration) *proto.RaidSimResult {
	return &proto.RaidSimResult{
		RaidMetrics: &proto.RaidMetrics{
			Dps:     &proto.DistributionMetrics{Avg: raidDpsAvg, AggregatorData: &proto.AggregatorData{}},
			Hps:     &proto.DistributionMetrics{AggregatorData: &proto.AggregatorData{}},
			Parties: []*proto.PartyMetrics{},
		},
		EncounterMetrics: &proto.EncounterMetrics{Targets: []*proto.UnitMetrics{}},
		IterationsDone:   iterationsDone,
		SampleIteration:  sample,
	}
}

// pickSampleIteration itself, in isolation: the documented rule is
// "closest to the combined mean," and on an exact tie the earlier shard in
// the slice wins (strict less-than never displaces an equal-distance
// incumbent). Both halves of the rule are pinned here so a future edit that
// changes either fails this test, not just a silent shift in production
// output.
func TestPickSampleIterationChoosesTheClosestToMean(t *testing.T) {
	far := &proto.SampleIteration{Dps: 80}       // |80-100| = 20
	closest := &proto.SampleIteration{Dps: 95}   // |95-100| = 5
	farthest := &proto.SampleIteration{Dps: 150} // |150-100| = 50

	results := []*proto.RaidSimResult{
		{SampleIteration: far},
		{SampleIteration: closest},
		{SampleIteration: farthest},
	}

	got := pickSampleIteration(results, 100)
	if got != closest {
		t.Fatalf("pickSampleIteration picked Dps=%v, want the closest-to-mean sample (Dps=%v)", got.Dps, closest.Dps)
	}
}

func TestPickSampleIterationBreaksTiesTowardTheEarlierShard(t *testing.T) {
	tiedFirst := &proto.SampleIteration{Dps: 90}   // |90-100| = 10
	tiedSecond := &proto.SampleIteration{Dps: 110} // |110-100| = 10, same distance

	results := []*proto.RaidSimResult{
		{SampleIteration: tiedFirst},
		{SampleIteration: tiedSecond},
	}

	got := pickSampleIteration(results, 100)
	if got != tiedFirst {
		t.Fatalf("pickSampleIteration on a tie picked Dps=%v, want the earlier shard's sample (Dps=%v)", got.Dps, tiedFirst.Dps)
	}
}

func TestPickSampleIterationSkipsShardsWithNoSample(t *testing.T) {
	only := &proto.SampleIteration{Dps: 95}
	results := []*proto.RaidSimResult{
		{SampleIteration: nil},
		{SampleIteration: only},
		{SampleIteration: nil},
	}

	got := pickSampleIteration(results, 100)
	if got != only {
		t.Fatalf("pickSampleIteration = %+v, want the one shard that carries a sample", got)
	}
}

func TestPickSampleIterationReturnsNilWhenNoShardSampled(t *testing.T) {
	results := []*proto.RaidSimResult{
		{SampleIteration: nil},
		{SampleIteration: nil},
	}

	if got := pickSampleIteration(results, 100); got != nil {
		t.Fatalf("pickSampleIteration = %+v, want nil when sampling was off on every shard", got)
	}
}

// End to end through the real entry point: several shards each carrying a
// sample. Each shard's RaidMetrics.Dps.Avg is 100 with equal IterationsDone,
// so the combined mean is exactly 100 (combineDistMetrics is a weighted sum
// of the shards' own Avg, and SetBaseResult contributes nothing to Avg on
// its own) - which makes the expected pick unambiguous and lets the
// assertion be exact rather than "somewhere in range."
func TestCombineConcurrentSimResultsPicksTheClosestShardSample(t *testing.T) {
	shard1 := syntheticShardResult(10, 100, &proto.SampleIteration{Dps: 80})
	shard2 := syntheticShardResult(10, 100, &proto.SampleIteration{Dps: 95})
	shard3 := syntheticShardResult(10, 100, &proto.SampleIteration{Dps: 150})

	combined := CombineConcurrentSimResults([]*proto.RaidSimResult{shard1, shard2, shard3}, false)

	// A three-way weighted split of an exact 100 isn't exactly representable
	// in binary floating point (100/3 repeats); a tight tolerance still
	// pins "all three shards contributed equally around 100" without
	// demanding bit-exactness a three-way division can't deliver.
	if !WithinToleranceFloat64(100, combined.RaidMetrics.Dps.Avg, 0.0001) {
		t.Fatalf("combined mean DPS = %v, want ~100 (fixture precondition)", combined.RaidMetrics.Dps.Avg)
	}
	if combined.SampleIteration != shard2.SampleIteration {
		t.Fatalf("combined.SampleIteration = %+v (Dps=%v), want shard2's exact sample (Dps=95, the closest to the mean)",
			combined.SampleIteration, combined.SampleIteration.Dps)
	}
}

// Only some shards carry a sample - e.g. a sample-iteration request combined
// with legacy or malformed shard data. The combine must still pick from
// among the shards that have one, not panic on the ones that don't.
func TestCombineConcurrentSimResultsPicksFromPartiallySampledShards(t *testing.T) {
	shard1 := syntheticShardResult(10, 100, nil)
	shard2 := syntheticShardResult(10, 100, &proto.SampleIteration{Dps: 95})
	shard3 := syntheticShardResult(10, 100, nil)

	combined := CombineConcurrentSimResults([]*proto.RaidSimResult{shard1, shard2, shard3}, false)

	if combined.SampleIteration != shard2.SampleIteration {
		t.Fatalf("combined.SampleIteration = %+v, want the one shard's sample carried through", combined.SampleIteration)
	}
}

// Sampling off on every shard: the combined result must carry no sample,
// and combining must not panic just because SampleIteration is nil
// everywhere.
func TestCombineConcurrentSimResultsProducesNoSampleWhenNoneSampled(t *testing.T) {
	shard1 := syntheticShardResult(10, 100, nil)
	shard2 := syntheticShardResult(10, 100, nil)

	combined := CombineConcurrentSimResults([]*proto.RaidSimResult{shard1, shard2}, false)

	if combined.SampleIteration != nil {
		t.Fatalf("combined.SampleIteration = %+v, want nil when no shard sampled", combined.SampleIteration)
	}
}
