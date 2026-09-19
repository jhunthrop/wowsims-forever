package core

import (
	"sort"
	"testing"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// sampleTestSpellID identifies the one spell sampleTestAgent knows. It has
// no real-world meaning; it only needs to be an ActionID the APL rotation
// below can address.
const sampleTestSpellID = 900001

// sampleTestAgent is a minimal caster with a real, repeating GCD action -
// package core cannot import a real class package (every class package
// imports core, so the reverse import would cycle), and the only other
// spec already registered for core's own test binary (dot_test.go's
// FakeAgent, for Elemental Shaman) never casts on its own: its
// OnGCDReady/Rotation stays nil, so nothing in the ordinary run loop ever
// calls it. This agent exists so the sample-iteration tests can exercise a
// real, ordered cast log rather than an empty one.
type sampleTestAgent struct {
	Character
	spell *Spell
}

func (a *sampleTestAgent) GetCharacter() *Character { return &a.Character }

func (a *sampleTestAgent) Initialize() {
	a.spell = a.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: sampleTestSpellID},
		SpellSchool:      SpellSchoolFire,
		ProcMask:         ProcMaskSpellDamage,
		Flags:            SpellFlagIgnoreResists,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Cast: CastConfig{
			DefaultCast: Cast{
				GCD: GCDDefault,
			},
		},
		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			baseDamage := sim.Roll(50, 150)
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeAlwaysHit)
		},
	})
}

func (a *sampleTestAgent) ApplyTalents()       {}
func (a *sampleTestAgent) Reset(_ *Simulation) {}

func newSampleTestAgent(character *Character, _ *proto.Player) Agent {
	return &sampleTestAgent{Character: *character}
}

func init() {
	RegisterAgentFactory(
		proto.Player_Mage{},
		proto.Spec_SpecMage,
		newSampleTestAgent,
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_Mage)
			if !ok {
				panic("Invalid spec value for sample test agent!")
			}
			player.Spec = playerSpec
		},
	)
}

// sampleTestRotation casts sampleTestAgent's one spell on cooldown. Every
// class's real APL is built the same way, from proto.APLAction values; this
// is the smallest one that keeps the GCD action - and so the whole run
// loop - genuinely occupied.
var sampleTestRotation = &proto.APLRotation{
	Type: proto.APLRotation_TypeAPL,
	PriorityList: []*proto.APLListItem{
		{
			Action: &proto.APLAction{
				Action: &proto.APLAction_CastSpell{
					CastSpell: &proto.APLActionCastSpell{
						SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: sampleTestSpellID}},
					},
				},
			},
		},
	},
}

func sampleRequest(iterations int32, sample bool) *proto.RaidSimRequest {
	return &proto.RaidSimRequest{
		Raid: SinglePlayerRaidProto(&proto.Player{
			Name:      "Sample Test",
			Race:      proto.Race_RaceOrc,
			Class:     proto.Class_ClassMage,
			Spec:      &proto.Player_Mage{Mage: &proto.Mage{}},
			Equipment: &proto.EquipmentSpec{},
			Rotation:  sampleTestRotation,
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{
			Iterations:      iterations,
			IsTest:          true,
			RandomSeed:      1,
			SampleIteration: sample,
		},
	}
}

// Off by default: a run that does not ask for a sample must not pay for
// one, and must not carry one.
func TestNoSampleWhenNotAsked(t *testing.T) {
	result := RunSim(sampleRequest(20, false), nil, simsignals.CreateSignals())
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	if result.SampleIteration != nil {
		t.Errorf("SampleIteration = %+v, want nil", result.SampleIteration)
	}
}

// The sample exists, has casts, and its casts are in time order.
func TestSampleIterationIsAnOrderedCastLog(t *testing.T) {
	result := RunSim(sampleRequest(20, true), nil, simsignals.CreateSignals())
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	sample := result.SampleIteration
	if sample == nil {
		t.Fatal("SampleIteration = nil, want a log")
	}
	if len(sample.Casts) == 0 {
		t.Fatal("SampleIteration has no casts")
	}
	for i := 1; i < len(sample.Casts); i++ {
		if sample.Casts[i].AtMs < sample.Casts[i-1].AtMs {
			t.Fatalf("cast %d at %dms precedes cast %d at %dms", i, sample.Casts[i].AtMs, i-1, sample.Casts[i-1].AtMs)
		}
	}
	if sample.DurationSeconds <= 0 {
		t.Errorf("DurationSeconds = %v, want the sampled iteration's length", sample.DurationSeconds)
	}
}

// The sampled iteration is the median one, not the first and not the
// luckiest: its DPS must sit inside the run's min-max band.
func TestSampledIterationIsNearTheMiddle(t *testing.T) {
	result := RunSim(sampleRequest(51, true), nil, simsignals.CreateSignals())
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	dps := result.RaidMetrics.Dps
	got := result.SampleIteration.Dps
	if got < dps.Min || got > dps.Max {
		t.Errorf("sample DPS %.1f is outside the run's [%.1f, %.1f]", got, dps.Min, dps.Max)
	}
}

// The extra iteration must not be counted: a 20-iteration run reports 20.
func TestSampleDoesNotPolluteTheAggregates(t *testing.T) {
	with := RunSim(sampleRequest(20, true), nil, simsignals.CreateSignals())
	without := RunSim(sampleRequest(20, false), nil, simsignals.CreateSignals())
	if with.Error != nil || without.Error != nil {
		t.Fatal("sim failed")
	}
	if with.IterationsDone != 20 {
		t.Errorf("IterationsDone = %d with sampling on, want 20", with.IterationsDone)
	}
	if with.RaidMetrics.Dps.Avg != without.RaidMetrics.Dps.Avg {
		t.Errorf("mean DPS with sampling %.6f differs from without %.6f; the replay leaked into the aggregates",
			with.RaidMetrics.Dps.Avg, without.RaidMetrics.Dps.Avg)
	}
}

// An iteration is reproducible from its seed alone, which is the whole
// premise of the replay.
func TestIterationSeedIsDerivedFromTheIndex(t *testing.T) {
	sim := NewSim(sampleRequest(10, false), simsignals.CreateSignals())
	if got := sim.iterationSeed(3); got != sim.Options.RandomSeed+3 {
		t.Errorf("iterationSeed(3) = %d, want %d", got, sim.Options.RandomSeed+3)
	}
	if got := sim.iterationSeed(0); got != sim.rseed {
		t.Errorf("iterationSeed(0) = %d, want the constructed seed %d", got, sim.rseed)
	}
}

// TestSampleDoesNotPolluteTheAggregates above only pins mean DPS. This test
// pins the WHOLE result: every field RunSim can produce, other than
// SampleIteration itself, must be identical whether or not sampling is on.
// That is the strongest form of "the replay never touches the reported
// metrics" - not close, not the same to six decimals, but proto.Equal.
func TestSamplingLeavesTheWholeResultByteIdentical(t *testing.T) {
	with := RunSim(sampleRequest(20, true), nil, simsignals.CreateSignals())
	without := RunSim(sampleRequest(20, false), nil, simsignals.CreateSignals())
	if with.Error != nil || without.Error != nil {
		t.Fatal("sim failed")
	}
	if with.SampleIteration == nil {
		t.Fatal("SampleIteration = nil with sampling on, want a log (precondition for this test)")
	}

	// SampleIteration is the one field sampling is allowed to add.
	with.SampleIteration = nil
	if !googleProto.Equal(with, without) {
		t.Errorf("RaidSimResult with sampling (minus SampleIteration) differs from without:\nwith:    %s\nwithout: %s",
			with.String(), without.String())
	}
}

// The sample is not merely "inside the band" (TestSampledIterationIsNearTheMiddle
// already covers that black-box); it is the run's exact median, and the
// engine reaches it purely by replaying that iteration's recorded seed on a
// fresh Simulation. This test opens up the package-private buffer to prove
// both halves directly: the DPS matches a from-scratch median computed over
// every recorded (dps, seed) pair, and re-running the main sim's own
// mechanics at the recorded seed - on yet another, independent Simulation -
// reproduces that exact DPS.
func TestSampleIsExactlyTheMedianAndItsSeedReproducesIt(t *testing.T) {
	req := sampleRequest(21, true)
	signals := simsignals.CreateSignals()

	sim := NewSim(req, signals)
	result := sim.run()
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	if len(sim.iterationSamples) != 21 {
		t.Fatalf("iterationSamples has %d entries, want 21", len(sim.iterationSamples))
	}

	dpsValues := make([]float64, len(sim.iterationSamples))
	for i, s := range sim.iterationSamples {
		dpsValues[i] = s.dps
	}
	sort.Float64s(dpsValues)
	wantMedianDPS := dpsValues[len(dpsValues)/2]

	sample := runSampleIteration(req, sim.iterationSamples, sim.BaseDuration, signals)
	if sample == nil {
		t.Fatal("runSampleIteration returned nil")
	}
	if sample.Dps != wantMedianDPS {
		t.Errorf("sample.Dps = %v, want the exact median %v", sample.Dps, wantMedianDPS)
	}

	var medianSeed int64
	found := false
	for _, s := range sim.iterationSamples {
		if s.dps == wantMedianDPS {
			medianSeed = s.seed
			found = true
			break
		}
	}
	if !found {
		t.Fatal("could not locate the median sample's own seed")
	}

	// Reproduce the iteration a THIRD way: a brand new Simulation, reseeded
	// to exactly the median's recorded seed, using the plain runOnce() path
	// (not runSampleIteration). If the seed alone did not determine the
	// iteration, this would not match.
	verify := NewSim(req, signals)
	verifyUnit := verify.sampleUnit()
	if verifyUnit == nil {
		t.Fatal("verify sim has no sample unit")
	}
	verify.reseedTo(medianSeed)
	verify.runOnce()
	if got := verifyUnit.Metrics.dps.LastIterationValue; got != wantMedianDPS {
		t.Errorf("re-running at the recorded seed produced DPS %v, want the exact median %v - the seed does not reproduce the iteration",
			got, wantMedianDPS)
	}
}
