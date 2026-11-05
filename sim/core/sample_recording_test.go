package core

import (
	"math"
	"testing"
	"time"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	recordingSpellID      = 900101
	recordingSelfBuffID   = 900102
	recordingTargetAuraID = 900103
)

// recordingAgent gives the sample recorder everything it has to see in one
// cast: a direct result that can miss and crit, a dot that ticks and crits, a
// buff on the caster, and a stacking debuff on the target.
type recordingAgent struct {
	Character
	spell      *Spell
	selfBuff   *Aura
	targetAura AuraArray
}

func (a *recordingAgent) GetCharacter() *Character { return &a.Character }

func (a *recordingAgent) Initialize() {
	a.selfBuff = a.RegisterAura(Aura{
		Label:    "Recording Self Buff",
		ActionID: ActionID{SpellID: recordingSelfBuffID},
		Duration: time.Second * 5,
	})
	a.targetAura = a.NewEnemyAuraArray(func(target *Unit) *Aura {
		return target.RegisterAura(Aura{
			Label:     "Recording Target Aura",
			ActionID:  ActionID{SpellID: recordingTargetAuraID},
			Duration:  time.Second * 8,
			MaxStacks: 3,
		})
	})
	a.spell = a.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: recordingSpellID},
		DefenseType:      DefenseTypeMagic,
		SpellSchool:      SpellSchoolFire,
		ProcMask:         ProcMaskSpellDamage,
		Flags:            SpellFlagIgnoreResists,
		BonusCritRating:  30 * CritRatingPerCritChance,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Cast: CastConfig{
			DefaultCast: Cast{GCD: time.Second * 6},
		},
		Dot: DotConfig{
			Aura:          Aura{Label: "Recording Dot"},
			NumberOfTicks: 4,
			TickLength:    time.Second * 3,
			CanCrit:       true,
			OnSnapshot: func(_ *Simulation, target *Unit, dot *Dot, isRollover bool) {
				dot.Snapshot(target, 40, isRollover)
			},
			OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},
		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			result := spell.CalcAndDealDamage(sim, target, sim.Roll(50, 150), spell.OutcomeMagicHitAndCrit)
			if result.Landed() {
				spell.Dot(target).Apply(sim)
			}
			a.selfBuff.Activate(sim)
			debuff := a.targetAura.Get(target)
			debuff.Activate(sim)
			debuff.AddStack(sim)
		},
	})
}

func (a *recordingAgent) ApplyTalents()       {}
func (a *recordingAgent) Reset(_ *Simulation) {}

func init() {
	RegisterAgentFactory(
		proto.Player_Warlock{},
		proto.Spec_SpecWarlock,
		func(character *Character, _ *proto.Player) Agent {
			return &recordingAgent{Character: *character}
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_Warlock)
			if !ok {
				panic("Invalid spec value for the recording test agent!")
			}
			player.Spec = playerSpec
		},
	)
}

func recordingRequest(iterations int32, sample bool, debuffs *proto.Debuffs) *proto.RaidSimRequest {
	return &proto.RaidSimRequest{
		Raid: SinglePlayerRaidProto(&proto.Player{
			Name:      "Recording Test",
			Race:      proto.Race_RaceOrc,
			Class:     proto.Class_ClassWarlock,
			Spec:      &proto.Player_Warlock{Warlock: &proto.Warlock{}},
			Equipment: &proto.EquipmentSpec{},
			Rotation: &proto.APLRotation{
				Type: proto.APLRotation_TypeAPL,
				PriorityList: []*proto.APLListItem{{
					Action: &proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{
						SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: recordingSpellID}},
					}}},
				}},
			},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, debuffs),
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

// replayOnce runs one iteration at a fixed seed, with or without a recorder,
// and returns the metrics it produced.
func replayOnce(req *proto.RaidSimRequest, record bool) (*proto.RaidMetrics, *proto.EncounterMetrics, *sampleRecorder) {
	sim := NewSim(req, simsignals.CreateSignals())
	if record {
		sim.sample = &sampleRecorder{unit: sim.sampleUnit()}
	}
	sim.reseedTo(7)
	sim.runOnce()
	return sim.Raid.GetMetrics(), sim.Encounter.GetMetricsProto(), sim.sample
}

// The recorder only reads: the same seed gives the same fight, to the last
// bit of every metric, with and without it.
func TestSampleRecorderLeavesTheFightUntouched(t *testing.T) {
	req := recordingRequest(1, true, &proto.Debuffs{})
	withRaid, withEncounter, recorder := replayOnce(req, true)
	withoutRaid, withoutEncounter, _ := replayOnce(req, false)

	if len(recorder.casts) == 0 || len(recorder.auras) == 0 {
		t.Fatalf("the recorder saw %d rows and %d aura events, want both", len(recorder.casts), len(recorder.auras))
	}
	if !googleProto.Equal(withRaid, withoutRaid) {
		t.Errorf("raid metrics moved with the recorder on:\nwith:    %s\nwithout: %s", withRaid, withoutRaid)
	}
	if !googleProto.Equal(withEncounter, withoutEncounter) {
		t.Errorf("encounter metrics moved with the recorder on:\nwith:    %s\nwithout: %s", withEncounter, withoutEncounter)
	}
}

// The whole result, new fields included, is the same with a sample as without.
func TestSampleRecordingLeavesTheWholeResultByteIdentical(t *testing.T) {
	with := RunSim(recordingRequest(40, true, &proto.Debuffs{}), nil, simsignals.CreateSignals())
	without := RunSim(recordingRequest(40, false, &proto.Debuffs{}), nil, simsignals.CreateSignals())
	if with.Error != nil || without.Error != nil {
		t.Fatal("sim failed")
	}
	if with.SampleIteration == nil {
		t.Fatal("no sample")
	}
	with.SampleIteration = nil
	if !googleProto.Equal(with, without) {
		t.Errorf("result differs with sampling on:\nwith:    %s\nwithout: %s", with, without)
	}
}

// The replay reproduces the median iteration exactly with the recorder running.
func TestSampleReplayStillReachesTheMedian(t *testing.T) {
	result := RunSim(recordingRequest(21, true, &proto.Debuffs{}), nil, simsignals.CreateSignals())
	sample := result.SampleIteration
	var total float64
	for _, row := range sample.Casts {
		total += row.Amount
	}
	if got := total / sample.DurationSeconds; math.Abs(got-sample.Dps) > 1e-6 {
		t.Errorf("the sample's own damage reads %.6f DPS, the median iteration %.6f", got, sample.Dps)
	}
}

// Every damage event is one row with its own outcome, counted the way the
// engine's metrics count it, and the rows sum to the iteration's damage.
func TestSampleRowsCarryOutcomeAndAmount(t *testing.T) {
	raid, _, recorder := replayOnce(recordingRequest(1, true, &proto.Debuffs{}), true)

	counts := map[proto.SampleOutcome]int32{}
	var damage float64
	for _, row := range recorder.casts {
		counts[row.Outcome]++
		damage += row.Amount
		landed := row.Outcome != proto.SampleOutcome_SampleOutcomeMiss
		if landed != (row.Amount > 0) {
			t.Errorf("row %+v: outcome and amount disagree", row)
		}
	}

	var target *proto.TargetedActionMetrics
	for _, action := range raid.Parties[0].Players[0].Actions {
		if action.Id.GetSpellId() == recordingSpellID {
			target = action.Targets[0]
		}
	}
	if target == nil {
		t.Fatal("no metrics for the recording spell")
	}
	if damage != target.Damage {
		t.Errorf("rows sum to %v damage, the metrics say %v", damage, target.Damage)
	}
	want := map[proto.SampleOutcome]int32{
		proto.SampleOutcome_SampleOutcomeHit:      target.Hits,
		proto.SampleOutcome_SampleOutcomeCrit:     target.Crits,
		proto.SampleOutcome_SampleOutcomeMiss:     target.Misses,
		proto.SampleOutcome_SampleOutcomeTick:     target.Ticks,
		proto.SampleOutcome_SampleOutcomeTickCrit: target.CritTicks,
	}
	for outcome, n := range want {
		if counts[outcome] != n {
			t.Errorf("%s rows = %d, the metrics count %d", outcome, counts[outcome], n)
		}
	}
	if want[proto.SampleOutcome_SampleOutcomeCrit] == 0 || want[proto.SampleOutcome_SampleOutcomeTick] == 0 || want[proto.SampleOutcome_SampleOutcomeTickCrit] == 0 {
		t.Errorf("the fight proved too little: %+v", want)
	}
}

// Each aura's on and off events, on the player and on the target, rebuild the
// uptime the engine reports.
func TestSampleAuraEventsRebuildTheUptime(t *testing.T) {
	raid, encounter, recorder := replayOnce(recordingRequest(1, true, &proto.Debuffs{}), true)

	uptime := func(spellID int32, wantTarget string) float64 {
		var total, startedAt int64
		up := false
		var maxStacks int32
		for _, e := range recorder.auras {
			if e.AuraId.GetSpellId() != spellID {
				continue
			}
			if e.Target != wantTarget {
				t.Errorf("aura %d event on %q, want %q", spellID, e.Target, wantTarget)
			}
			maxStacks = max(maxStacks, e.Stacks)
			if e.Active && !up {
				startedAt, up = e.AtMs, true
			} else if !e.Active && up {
				total += e.AtMs - startedAt
				up = false
			}
		}
		if spellID == recordingTargetAuraID && maxStacks != 3 {
			t.Errorf("target aura stacks reached %d, want 3", maxStacks)
		}
		if up {
			t.Errorf("aura %d never faded in the log", spellID)
		}
		return float64(total) / 1000
	}

	metricUptime := func(auras []*proto.AuraMetrics, spellID int32) float64 {
		for _, m := range auras {
			if m.Id.GetSpellId() == spellID {
				return m.UptimeSecondsAvg
			}
		}
		t.Fatalf("no metrics for aura %d", spellID)
		return 0
	}

	if got, want := uptime(recordingSelfBuffID, ""), metricUptime(raid.Parties[0].Players[0].Auras, recordingSelfBuffID); math.Abs(got-want) > 0.002 {
		t.Errorf("player aura uptime from events = %v, metrics say %v", got, want)
	}
	if got, want := uptime(recordingTargetAuraID, "Target 1"), metricUptime(encounter.Targets[0].Auras, recordingTargetAuraID); math.Abs(got-want) > 0.002 {
		t.Errorf("target aura uptime from events = %v, metrics say %v", got, want)
	}
}

// The sheet is the engine's own stat measurement, read off a separate
// environment. Debuffs the request applies show on the target's armor.
func TestSampleSheetReadsTheCharacterAndTarget(t *testing.T) {
	req := recordingRequest(1, true, &proto.Debuffs{SunderArmor: true})
	armored := googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
	armored.Stats[stats.Armor] = 3731
	req.Encounter.Targets = []*proto.Target{armored}
	sheet := sampleSheet(req)
	if sheet == nil {
		t.Fatal("no sheet")
	}
	computed := ComputeStats(&proto.ComputeStatsRequest{Raid: req.Raid, Encounter: req.Encounter})
	final := computed.RaidStats.Parties[0].Players[0].FinalStats.Stats
	if sheet.AttackPower != final[int(stats.AttackPower)] {
		t.Errorf("AttackPower = %v, ComputeStats says %v", sheet.AttackPower, final[int(stats.AttackPower)])
	}
	if sheet.TargetLevel != 63 {
		t.Errorf("TargetLevel = %d, want 63", sheet.TargetLevel)
	}
	if sheet.TargetArmor != 3731 {
		t.Errorf("TargetArmor = %v, want the target's own 3731", sheet.TargetArmor)
	}
	// Five stacks of the rank a level 60 raid has learned: 450 a stack.
	if want := 3731.0 - 5*450; sheet.TargetArmorAfterDebuffs != want {
		t.Errorf("armor after five Sunders = %v, want %v", sheet.TargetArmorAfterDebuffs, want)
	}
	// Level 60 against a level 63 boss: 15 skill short is 3 crit points at 0.2
	// per point, plus the flat 1.8.
	wantCrit := (final[int(stats.Crit)] + final[int(stats.MeleeCrit)]) - 4.8
	if math.Abs(sheet.MeleeCritPctVsTarget-wantCrit) > 1e-9 {
		t.Errorf("MeleeCritPctVsTarget = %v, want %v", sheet.MeleeCritPctVsTarget, wantCrit)
	}
	if sheet.MeleeHastePct != 0 {
		t.Errorf("MeleeHastePct = %v, want none", sheet.MeleeHastePct)
	}
}

// The sample run carries the sheet.
func TestSampleIterationCarriesTheSheet(t *testing.T) {
	result := RunSim(recordingRequest(3, true, &proto.Debuffs{}), nil, simsignals.CreateSignals())
	if result.SampleIteration.GetSheet() == nil {
		t.Error("SampleIteration has no sheet")
	}
	if len(result.SampleIteration.GetAuras()) == 0 {
		t.Error("SampleIteration has no aura events")
	}
}

// Extra attacks are booked per source from the pull, and shards add up.
func TestExtraAttacksAreBookedPerSource(t *testing.T) {
	metrics := NewUnitMetrics()
	sim := &Simulation{}
	sim.CurrentTime = -time.Second
	source := ActionID{SpellID: 10610}
	metrics.addExtraAttacks(sim, source, 1)
	sim.CurrentTime = time.Second
	metrics.addExtraAttacks(sim, source, 2)
	metrics.addExtraAttacks(sim, ActionID{ItemID: 11815}, 1)

	got := metrics.extraAttacksProto()
	if len(got) != 2 {
		t.Fatalf("sources = %v, want 2", got)
	}
	counts := map[string]int32{}
	for _, row := range got {
		counts[row.Id.String()] = row.Attacks
	}
	if counts[source.ToProto().String()] != 2 {
		t.Errorf("Windfury = %v, want 2: the pre-pull grant stays out", counts)
	}

	combiner := raidSimResultCombiner{}
	base := &proto.UnitMetrics{}
	for range 2 {
		for _, row := range got {
			combiner.addExtraAttackMetrics(base, row)
		}
	}
	if len(base.ExtraAttacks) != 2 {
		t.Fatalf("combined sources = %v", base.ExtraAttacks)
	}
	for _, row := range base.ExtraAttacks {
		if row.Attacks != 2*counts[row.Id.String()] {
			t.Errorf("combined %s = %d, want double %d", row.Id, row.Attacks, counts[row.Id.String()])
		}
	}
}
