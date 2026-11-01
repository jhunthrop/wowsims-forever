package sim

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// raidDamageLog runs a shadow priest next to five fake raid members for
// twelve seconds, with the given damage model, and returns the debug log.
func raidDamageLog(t *testing.T, model *proto.RaidDamageModel) string {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassPriest, Race: proto.Race_RaceUndead, Level: 60,
		Equipment: &proto.EquipmentSpec{}, Rotation: &proto.APLRotation{},
	}, &proto.Player_ShadowPriest{ShadowPriest: &proto.ShadowPriest{Options: &proto.ShadowPriest_Options{}}})
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	raid.Parties = append(raid.Parties, &proto.Party{})
	raid.TargetDummies = 5
	raid.RaidDamageModel = model
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 12, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, IsTest: true, Debug: true, RandomSeed: 1},
	})
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	return result.Logs
}

func TestRaidDamageModelHurtsTheFakeRaid(t *testing.T) {
	logs := raidDamageLog(t, &proto.RaidDamageModel{
		Profile: "test", TankHealth: 9000, MemberHealth: 4500,
		TankHitDamage: 1500, TankSwingSeconds: 2, DamageSpread: 0.1,
		PulseDamage: 600, PulseIntervalSeconds: 5, PulseMembers: 2,
	})
	tankHits := strings.Count(logs, "[Target Dummy 5 (#6)] Spent")
	if tankHits < 5 {
		t.Errorf("the tank took %d hits in 12 seconds of 2 second swings, want at least 5", tankHits)
	}
	if !strings.Contains(logs, "[Target Dummy 1 (#2)] Spent") && !strings.Contains(logs, "[Target Dummy 2 (#3)] Spent") {
		t.Errorf("no pulse ever reached a raid member")
	}
	if strings.Contains(logs, "(0.000 --> ") {
		t.Errorf("a fake member kept taking damage at zero health")
	}
}

func TestNoRaidDamageModelLeavesTheFakeRaidAlone(t *testing.T) {
	if logs := raidDamageLog(t, nil); strings.Contains(logs, "Target Dummy") {
		t.Errorf("fake members took damage with no damage model")
	}
}

// pulsePriestHits counts the pulses the debug log shows landing on the
// real player (the priest, raid player 0) rather than a fake member.
func pulsePriestHits(logs string) int {
	return strings.Count(logs, "[ (#1)] Spent")
}

// A raid-wide pulse lands on the healer standing in the party as well as on
// the fake members, when the model says pulses reach the healer: a breath
// or a volley does not skip the player casting the heals.
func TestRaidDamagePulsesReachTheHealer(t *testing.T) {
	model := &proto.RaidDamageModel{
		Profile: "test", TankHealth: 9000, MemberHealth: 4500,
		PulseDamage: 600, PulseIntervalSeconds: 1, PulseMembers: 0,
		PulsesReachHealer: true,
	}
	if hits := pulsePriestHits(raidDamageLog(t, model)); hits < 10 {
		t.Errorf("the priest took %d pulses in 12 seconds of 1 second pulses that reach every member, want at least 10", hits)
	}
	model.PulsesReachHealer = false
	if hits := pulsePriestHits(raidDamageLog(t, model)); hits != 0 {
		t.Errorf("the priest took %d pulses with pulses_reach_healer off, want 0", hits)
	}
}
