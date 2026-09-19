package sim

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// The dungeon-pull timeline from the contract's section 1.6: one boss,
// then packs of three and five, then back down. Total damage must land
// between a pure single-target fight and a permanent five-target fight,
// because the adds are up for part of the time and not all of it.
func TestDungeonTimelineLandsBetweenOneAndFiveTargets(t *testing.T) {
	dungeon := []*proto.TargetCountAt{
		{AtSeconds: 0, Count: 1},
		{AtSeconds: 40, Count: 3},
		{AtSeconds: 80, Count: 5},
		{AtSeconds: 130, Count: 3},
		{AtSeconds: 160, Count: 1},
	}

	for _, tc := range []struct {
		name   string
		player func() *proto.Player
	}{
		{"fury warrior", furyWarriorPlayer},
		{"frost mage", frostMagePlayer},
	} {
		t.Run(tc.name, func(t *testing.T) {
			single := runParitySim(t, tc.player(), parityEncounter())

			five := parityEncounter()
			five.Targets = []*proto.Target{
				core.DefaultTargetProtoLvl60, core.DefaultTargetProtoLvl60, core.DefaultTargetProtoLvl60,
				core.DefaultTargetProtoLvl60, core.DefaultTargetProtoLvl60,
			}
			fiveDps := runParitySim(t, tc.player(), five)

			timeline := parityEncounter()
			timeline.TargetsOverTime = dungeon
			timelineDps := runParitySim(t, tc.player(), timeline)

			if timelineDps < single {
				t.Errorf("dungeon-timeline DPS %.1f is below the single-target DPS %.1f", timelineDps, single)
			}
			if timelineDps > fiveDps {
				t.Errorf("dungeon-timeline DPS %.1f is above the permanent five-target DPS %.1f", timelineDps, fiveDps)
			}
		})
	}
}

// The timeline's target count is what the rotation sees: an APL asking
// "how many targets" during the one-target opening must be told one, not
// the pool's five.
func TestNumTargetsFollowsTheTimeline(t *testing.T) {
	request := &proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(furyWarriorPlayer(), &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration: 180,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
			TargetsOverTime: []*proto.TargetCountAt{
				{AtSeconds: 0, Count: 1},
				{AtSeconds: 90, Count: 5},
			},
		},
		SimOptions: &proto.SimOptions{Iterations: 1, IsTest: true, RandomSeed: 1},
	}

	sim := core.NewSim(request, simsignals.CreateSignals())
	sim.Reset()
	if got := sim.GetNumTargets(); got != 1 {
		t.Fatalf("GetNumTargets at the pull = %d, want 1", got)
	}

	result := core.RunSim(request, nil, simsignals.CreateSignals())
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	if got := len(result.EncounterMetrics.Targets); got != 5 {
		t.Errorf("EncounterMetrics has %d targets, want the whole pool of 5 so the report can name the adds", got)
	}
}

// whirlwindSpellID is the warrior ability the capped-AoE tests below use:
// a hard cap of four swings, so it exercises both halves of the bound.
const whirlwindSpellID = 1680

// A capped AoE ability may swing at any one target at most once per cast.
// The hit loop is sized once, at spell-registration time, but it walks
// targets with NextTargetUnit, which wraps at the LIVE target count - so
// when a timeline shrinks the active prefix below the registration count
// the surplus swings used to wrap back onto the surviving mob and inflate
// its damage.
func TestShrinkingTimelineDoesNotRepeatHitsOnOneTarget(t *testing.T) {
	encounter := parityEncounter()
	encounter.TargetsOverTime = []*proto.TargetCountAt{
		{AtSeconds: 0, Count: 5},
		{AtSeconds: 1, Count: 1},
	}

	whirlwind := actionMetrics(t, runParityResult(t, furyWarriorPlayer(), encounter), whirlwindSpellID)

	var casts, swingsOnSurvivor int32
	for _, target := range whirlwind.Targets {
		casts += target.Casts
		if target.UnitIndex == 0 {
			swingsOnSurvivor = swingCount(target)
		}
	}
	if casts == 0 {
		t.Fatal("the fury warrior cast no Whirlwind; the test cannot say anything")
	}
	if swingsOnSurvivor > casts {
		t.Errorf("Whirlwind swung %d times at the one surviving target over %d casts; a capped AoE may swing at a target at most once per cast",
			swingsOnSurvivor, casts)
	}
}

// The other half of the same bound: a timeline that grows past the count
// the pull started with must give a capped AoE its extra targets, up to
// the ability's own cap. Whirlwind's cap is four, so four of the five
// pooled targets must see swings.
func TestGrowingTimelineGivesCappedAoEItsTargets(t *testing.T) {
	encounter := parityEncounter()
	encounter.TargetsOverTime = []*proto.TargetCountAt{
		{AtSeconds: 0, Count: 1},
		{AtSeconds: 1, Count: 5},
	}

	whirlwind := actionMetrics(t, runParityResult(t, furyWarriorPlayer(), encounter), whirlwindSpellID)

	var swungAt int
	for _, target := range whirlwind.Targets {
		if swingCount(target) > 0 {
			swungAt++
		}
	}
	if swungAt < 4 {
		t.Errorf("Whirlwind swung at %d targets, want its cap of 4 once the timeline grew to five", swungAt)
	}
}

// Every pooled target must be finished at the end of an iteration, not
// just the active prefix: Encounter.doneIteration and GetMetricsProto
// either side of the loop both walk the whole pool, so a target that is
// never finished divides by zero and reports NaN DPS to the site.
func TestEveryPooledTargetReportsFiniteMetrics(t *testing.T) {
	encounter := parityEncounter()
	encounter.TargetsOverTime = []*proto.TargetCountAt{
		{AtSeconds: 0, Count: 1},
		{AtSeconds: 40, Count: 5},
		{AtSeconds: 160, Count: 1},
	}

	result := runParityResult(t, furyWarriorPlayer(), encounter)
	if got := len(result.EncounterMetrics.Targets); got != 5 {
		t.Fatalf("EncounterMetrics has %d targets, want the whole pool of 5", got)
	}
	for i, target := range result.EncounterMetrics.Targets {
		dps := target.GetDps().GetAvg()
		if math.IsNaN(dps) || math.IsInf(dps, 0) {
			t.Errorf("target %d reports dps.avg = %v; every pooled target must finish the iteration", i, dps)
		}
	}
}

// swingCount is every attempt this action made against one target,
// whatever the outcome. A capped AoE's attempts are what the hit loop
// bounds, so this is the number the two tests above compare.
func swingCount(target *proto.TargetedActionMetrics) int32 {
	return target.Hits + target.Crits + target.Misses + target.Dodges +
		target.Parries + target.Blocks + target.BlockedCrits + target.Glances + target.Crushes
}

// actionMetrics finds one spell's metrics in the first player's report.
func actionMetrics(t *testing.T, result *proto.RaidSimResult, spellID int32) *proto.ActionMetrics {
	t.Helper()
	player := result.RaidMetrics.Parties[0].Players[0]
	for _, action := range player.Actions {
		if action.Id.GetSpellId() == spellID {
			return action
		}
	}
	t.Fatalf("player %s has no metrics for spell %d", player.Name, spellID)
	return nil
}

// runParityResult is runParitySim's sibling for the tests that need the
// whole report rather than just the raid's DPS.
func runParityResult(t *testing.T, player *proto.Player, encounter *proto.Encounter) *proto.RaidSimResult {
	t.Helper()
	result := core.RunSim(&proto.RaidSimRequest{
		Raid:      core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: encounter,
		SimOptions: &proto.SimOptions{
			Iterations: parityIterations,
			IsTest:     true,
			RandomSeed: 1,
		},
	}, nil, simsignals.CreateSignals())
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	return result
}
