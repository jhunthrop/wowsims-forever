package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Forever puts Multi-Shot (2643) and every Aimed Shot rank in spell
// category 2 with a 6 s category cooldown (SpellCategories and
// SpellCooldowns, client 1.60.1.70658); both tooltips say the two "share
// a cooldown". Arcane Shot is category 1173 and keeps its own clock.
func TestMultiShotAndAimedShotShareOneCooldown(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()

	hunter := sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
	if hunter.MultiShot == nil || hunter.AimedShot == nil || hunter.ArcaneShot == nil {
		t.Fatal("level-60 hunter is missing Multi-Shot, Aimed Shot or Arcane Shot")
	}

	if hunter.MultiShot.CD.Timer != hunter.AimedShot.CD.Timer {
		t.Error("Multi-Shot and Aimed Shot run separate cooldown timers; the client shares category 2")
	}
	if hunter.ArcaneShot.CD.Timer == hunter.MultiShot.CD.Timer {
		t.Error("Arcane Shot shares the Multi-Shot timer; the client gives it its own category 1173")
	}
}
