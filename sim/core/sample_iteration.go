package core

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// sampleRecorder collects one unit's casts for one iteration.
type sampleRecorder struct {
	unit  *Unit
	casts []*proto.SampleCast
}

// iterationSample is one (dps, seed) pair from the main run. Sixteen
// bytes an iteration is the whole memory cost of knowing which iteration
// was the median one.
type iterationSample struct {
	dps  float64
	seed int64
}

// recordIterationSample appends the just-finished iteration's (dps, seed)
// pair to the run's sample buffer. Both call sites in run() share this body;
// only the seed differs, since iteration 0 and every later iteration derive
// theirs differently (see iterationSeed).
func (sim *Simulation) recordIterationSample(sampleUnit *Unit, seed int64) {
	sim.iterationSamples = append(sim.iterationSamples, iterationSample{
		dps:  sampleUnit.Metrics.dps.LastIterationValue,
		seed: seed,
	})
}

// recordSampleCast appends one cast. Spell.applyEffects calls it after
// the effects have run, so the resources read are the ones the player saw
// after the cast - costs already spent, and any rage or energy the cast
// itself generated already added.
func (sim *Simulation) recordSampleCast(spell *Spell, target *Unit) {
	if sim.sample == nil || spell.Unit != sim.sample.unit {
		return
	}

	cast := &proto.SampleCast{
		AtMs:      sim.CurrentTime.Milliseconds(),
		ActionId:  spell.ActionID.ToProto(),
		Resources: unitResources(spell.Unit),
	}
	if target != nil {
		cast.Target = target.Label
	}
	sim.sample.casts = append(sim.sample.casts, cast)
}

// unitResources reads the bars the unit actually has. A mage has no rage
// bar and an empty key would read as "0 rage" on the report, which is a
// different claim from "this class has no rage".
func unitResources(unit *Unit) map[string]int32 {
	resources := make(map[string]int32, 4)
	if unit.HasManaBar() {
		resources["mana"] = int32(unit.CurrentMana())
	}
	if unit.HasRageBar() {
		resources["rage"] = int32(unit.CurrentRage())
	}
	if unit.HasEnergyBar() {
		resources["energy"] = int32(unit.CurrentEnergy())
		resources["combo_points"] = unit.ComboPoints()
	}
	return resources
}

// sampleUnit is the unit whose casts the log is of: the first player in
// the raid. Every sim the site sends is a single character, and a raid
// sim's sample log has no single subject to be about.
func (sim *Simulation) sampleUnit() *Unit {
	for _, party := range sim.Raid.Parties {
		for _, player := range party.Players {
			return &player.GetCharacter().Unit
		}
	}
	return nil
}

// runSampleIteration replays the median-DPS iteration of a finished run.
//
// It builds a FRESH Simulation from the same request rather than reusing
// the finished one, because runOnce ends in Cleanup, which feeds
// doneIteration, which would add a twenty-first sample to a
// twenty-iteration run's aggregates. A fresh environment costs one
// construction and buys exact aggregates.
func runSampleIteration(rsr *proto.RaidSimRequest, samples []iterationSample, baseDuration time.Duration, signals simsignals.Signals) *proto.SampleIteration {
	if len(samples) == 0 {
		return nil
	}

	ordered := slices.Clone(samples)
	slices.SortStableFunc(ordered, func(a, b iterationSample) int {
		switch {
		case a.dps < b.dps:
			return -1
		case a.dps > b.dps:
			return 1
		default:
			return 0
		}
	})
	median := ordered[len(ordered)/2]

	sim := NewSim(rsr, signals)
	unit := sim.sampleUnit()
	if unit == nil {
		return nil
	}

	// A health fight's duration is estimated from the presim; carry the
	// finished run's estimate over so the replay is the same fight.
	sim.BaseDuration = baseDuration
	sim.Encounter.DurationIsEstimate = false

	sim.sample = &sampleRecorder{unit: unit}
	sim.reseedTo(median.seed)
	sim.runOnce()

	return &proto.SampleIteration{
		Dps:             median.dps,
		DurationSeconds: sim.Duration.Seconds(),
		Casts:           sim.sample.casts,
	}
}
