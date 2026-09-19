package sim

import (
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
