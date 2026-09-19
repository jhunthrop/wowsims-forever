package sim

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// fullDebuffs is the raid debuff panel a normal fight carries and a dummy
// does not.
func fullDebuffs() *proto.Debuffs {
	return &proto.Debuffs{
		SunderArmor:     true,
		FaerieFire:      true,
		CurseOfElements: true,
	}
}

func runDummyParitySim(t *testing.T, player *proto.Player, encounter *proto.Encounter) float64 {
	t.Helper()
	result := core.RunSim(&proto.RaidSimRequest{
		Raid:      core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, fullDebuffs()),
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
	return result.RaidMetrics.Dps.Avg
}

// A dummy has none of the raid's debuffs and no execute window, so both
// reference specs do less damage on one than on a real boss.
func TestDummyModeIsWorseThanARealBossForBothSpecs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		player func() *proto.Player
	}{
		{"fury warrior", furyWarriorPlayer},
		{"frost mage", frostMagePlayer},
	} {
		t.Run(tc.name, func(t *testing.T) {
			boss := parityEncounter()
			boss.ExecuteProportion_20 = 0.25
			bossDps := runDummyParitySim(t, tc.player(), boss)

			dummy := parityEncounter()
			dummy.ExecuteProportion_20 = 0.25
			dummy.TargetDummy = true
			dummyDps := runDummyParitySim(t, tc.player(), dummy)

			if dummyDps >= bossDps {
				t.Errorf("dummy DPS %.1f is not below the boss DPS %.1f", dummyDps, bossDps)
			}
		})
	}
}

// The execute half of the claim, isolated: a fury warrior's Execute is
// the largest single thing the window buys, so on a dummy the spell must
// never be cast at all.
func TestDummyModeNeverLetsTheWarriorExecute(t *testing.T) {
	dummy := parityEncounter()
	dummy.ExecuteProportion_20 = 1.0
	dummy.TargetDummy = true

	result := core.RunSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(furyWarriorPlayer(), &proto.PartyBuffs{}, &proto.RaidBuffs{}, fullDebuffs()),
		Encounter:  dummy,
		SimOptions: &proto.SimOptions{Iterations: parityIterations, IsTest: true, RandomSeed: 1},
	}, nil, simsignals.CreateSignals())
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}

	for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
		if action.Id.GetSpellId() == 0 {
			continue
		}
		casts := int32(0)
		for _, target := range action.Targets {
			casts += target.Casts
		}
		if casts > 0 && isWarriorExecute(action.Id.GetSpellId()) {
			t.Errorf("Execute (spell %d) was cast %d times on a target dummy", action.Id.GetSpellId(), casts)
		}
	}
}

// isWarriorExecute names the Execute ranks the client ships, so the test
// does not depend on which rank the reference gear affords.
func isWarriorExecute(spellID int32) bool {
	switch spellID {
	case 5308, 20658, 20660, 20661, 20662:
		return true
	}
	return false
}
