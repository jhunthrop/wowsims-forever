package sim

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// raidDamageLog runs a shadow priest next to five fake raid members for
// twelve seconds, with the given damage model, and returns the debug log.
func raidDamageLog(t *testing.T, model *proto.RaidDamageModel) string {
	t.Helper()
	return raidDamageLogWith(t, model, &proto.APLRotation{})
}

// raidDamageLogWith is raidDamageLog with the priest running rotation.
func raidDamageLogWith(t *testing.T, model *proto.RaidDamageModel, rotation *proto.APLRotation) string {
	t.Helper()
	return raidDamageLogWithTalents(t, model, rotation, "")
}

// raidDamageLogWithTalents is raidDamageLogWith with a talent string.
func raidDamageLogWithTalents(t *testing.T, model *proto.RaidDamageModel, rotation *proto.APLRotation, talents string) string {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassPriest, Race: proto.Race_RaceUndead, Level: 60,
		Equipment: &proto.EquipmentSpec{}, Rotation: rotation, TalentsString: talents,
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

// A raid-wide pulse that lands on the healer mid-cast costs casting time,
// as any damage does: the breath or volley the pulse stands for pushes a
// cast back like a hit from a spell. Smite (2.5 s) under a pulse every
// second must be pushed back.
func TestRaidDamagePulsePushesBackTheHealersCast(t *testing.T) {
	model := &proto.RaidDamageModel{
		Profile: "test", TankHealth: 9000, MemberHealth: 4500,
		PulseDamage: 10, PulseIntervalSeconds: 1, PulseMembers: 0,
		PulsesReachHealer: true,
	}
	smite := core.APLRotationFromJsonString(`{"type": "TypeAPL", "priorityList": [{"action": {"castSpell": {"spellId": {"spellId": 10934}}}}]}`)
	logs := raidDamageLogWith(t, model, smite)
	if !strings.Contains(logs, "Smite") && !strings.Contains(logs, "10934") {
		t.Fatalf("the priest never cast Smite; the test rotation is wrong")
	}
	if n := strings.Count(logs, "Hardcast extended by"); n < 3 {
		t.Errorf("the priest's casts were pushed back %d times under a pulse a second, want at least 3", n)
	}
	model.PulsesReachHealer = false
	if n := strings.Count(raidDamageLogWith(t, model, smite), "Hardcast extended by"); n != 0 {
		t.Errorf("casts were pushed back %d times with no pulse reaching the healer, want 0", n)
	}
}

// A channel hit by a pulse loses duration instead of being delayed: Mind
// Flay (3 s, a tick a second) under a pulse every half second must be cut
// short, and the priest must go on casting after it.
func TestRaidDamagePulseShortensTheHealersChannel(t *testing.T) {
	model := &proto.RaidDamageModel{
		Profile: "test", TankHealth: 9000, MemberHealth: 4500,
		PulseDamage: 10, PulseIntervalSeconds: 0.5, PulseMembers: 0,
		PulsesReachHealer: true,
	}
	flay := core.APLRotationFromJsonString(`{"type": "TypeAPL", "priorityList": [{"action": {"castSpell": {"spellId": {"spellId": 18807}}}}]}`)
	logs := raidDamageLogWithTalents(t, model, flay, "--00000001")
	casts := strings.Count(logs, "Casting {SpellID: 18807}")
	if casts == 0 {
		casts = strings.Count(logs, "Completed cast {SpellID: 18807}")
	}
	if n := strings.Count(logs, "Channel shortened by"); n < 3 {
		t.Errorf("Mind Flay was shortened %d times under a pulse every half second, want at least 3", n)
	}
	if casts < 5 {
		t.Errorf("the priest cast Mind Flay %d times in 12 seconds of shortened channels, want at least 5 (the rotation stalled)", casts)
	}
}

// A cast is never pushed back past its own full cast time from now (the
// vanilla server caps the delay there: cmangos-classic Spell::Delayed), so
// under a pulse every quarter second no single delay is longer than the
// time since the cast began or was last pushed.
func TestPushbackNeverRestartsMoreThanTheFullCast(t *testing.T) {
	model := &proto.RaidDamageModel{
		Profile: "test", TankHealth: 9000, MemberHealth: 4500,
		PulseDamage: 1, PulseIntervalSeconds: 0.25, PulseMembers: 0,
		PulsesReachHealer: true,
	}
	smite := core.APLRotationFromJsonString(`{"type": "TypeAPL", "priorityList": [{"action": {"castSpell": {"spellId": {"spellId": 10934}}}}]}`)
	logs := raidDamageLogWith(t, model, smite)
	delays := regexp.MustCompile(`Hardcast extended by ([0-9.]+)(m?s) due`).FindAllStringSubmatch(logs, -1)
	if len(delays) == 0 {
		t.Fatal("no cast was pushed back")
	}
	for _, d := range delays {
		seconds, _ := strconv.ParseFloat(d[1], 64)
		if d[2] == "ms" {
			seconds /= 1000
		}
		if seconds > 0.26 {
			t.Errorf("a cast was pushed back %s%s under a pulse every 0.25 s; the delay must stop at the full cast time", d[1], d[2])
			break
		}
	}
}
