package elemental

import (
	"os"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/shaman"
)

// raidBuildTalents is the published Elemental raid build (31 Elemental with
// Lightning Mastery), the engine string of the guide's raidBuild.
const raidBuildTalents = "5500331220001351-055-053002"

// Under Lightning Mastery a rank 10 Lightning Bolt is a 1.5 s cast (2.5 s,
// less Elemental Alacrity's 0.5 and Mastery's 0.5), already on the 1.5 s
// global cooldown that casting speed never shortens, so Troll's Berserking
// inside it is wasted. The forever APL presses Berserking only once Mastery
// has ended; the round 26 guide audit found the engine firing both on the
// pull through "autocast other cooldowns".
func TestTrollElementalNeverOverlapsBerserkingWithLightningMastery(t *testing.T) {
	raw, err := os.ReadFile("../../../ui/elemental_shaman/apls/forever_elemental.apl.json")
	if err != nil {
		t.Fatal(err)
	}
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassShaman,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      raidBuildTalents,
			DistanceFromTarget: 20,
			Rotation:           core.APLRotationFromJsonString(string(raw)),
		},
		PlayerOptionsAdaptive,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	s := agent.GetShaman()
	mastery, berserking := s.GetAura("Lightning Mastery"), s.GetAura("Berserking")
	if mastery == nil || berserking == nil {
		t.Fatal("the raid build Troll lacks Lightning Mastery or Berserking")
	}

	var masteryAt, berserkingAt time.Duration = -1, -1
	for sim.CurrentTime < 40*time.Second {
		if done := sim.Step(); done {
			break
		}
		if mastery.IsActive() && masteryAt < 0 {
			masteryAt = sim.CurrentTime
		}
		if berserking.IsActive() && berserkingAt < 0 {
			berserkingAt = sim.CurrentTime
		}
		if mastery.IsActive() && berserking.IsActive() {
			t.Fatalf("Berserking is up inside Lightning Mastery at %v", sim.CurrentTime)
		}
	}
	if masteryAt < 0 || berserkingAt < 0 {
		t.Fatalf("Lightning Mastery at %v, Berserking at %v: want both pressed in the first 40 s", masteryAt, berserkingAt)
	}
	if berserkingAt < masteryAt+15*time.Second {
		t.Errorf("Berserking at %v, want once Mastery (pressed at %v) has run its 15 s", berserkingAt, masteryAt)
	}
}
