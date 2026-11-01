package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/rogue"
)

func buildRogueWithConsumes(t *testing.T, consumes *proto.Consumes) *rogue.Rogue {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassRogue,
			Race:      proto.Race_RaceOrc,
			Level:     60,
			Equipment: testDaggers,
			Consumes:  consumes,
			Rotation:  &proto.APLRotation{},
		},
		DefaultRogue,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim.Raid.Parties[0].Players[0].(rogue.RogueAgent).GetRogue()
}

// Forever lets a sharpening stone share a weapon with a rogue poison (the
// rogue and warlock Deep Dive, 2026-10-07). Each hand carries the poison
// in its imbue field and the stone in its stone field, and both apply:
// Dense Sharpening Stone adds 8 to the weapon's damage beside the poison.
func TestDenseStoneAppliesBesideAPoison(t *testing.T) {
	poisonsOnly := buildRogueWithConsumes(t, &proto.Consumes{
		MainHandImbue: proto.WeaponImbue_DeadlyPoison,
		OffHandImbue:  proto.WeaponImbue_InstantPoison,
	})
	withStones := buildRogueWithConsumes(t, &proto.Consumes{
		MainHandImbue: proto.WeaponImbue_DeadlyPoison,
		OffHandImbue:  proto.WeaponImbue_InstantPoison,
		MainHandStone: proto.WeaponImbue_DenseSharpeningStone,
		OffHandStone:  proto.WeaponImbue_DenseSharpeningStone,
	})
	for _, hand := range []struct {
		name       string
		base, with *core.Weapon
	}{
		{"main hand", poisonsOnly.AutoAttacks.MH(), withStones.AutoAttacks.MH()},
		{"off hand", poisonsOnly.AutoAttacks.OH(), withStones.AutoAttacks.OH()},
	} {
		if got := hand.with.BaseDamageMin - hand.base.BaseDamageMin; got != 8 {
			t.Errorf("%s: stone beside the poison adds %.1f minimum damage, want 8", hand.name, got)
		}
	}
	if !withStones.HasDagger(core.MainHand) {
		t.Fatal("test rogue is not wearing daggers")
	}
}

// The stone field holds a stone, weightstone or oil, never a second poison
// or a shaman imbue: that is a configuration error, refused at build time.
func TestStoneFieldRefusesAPoison(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a poison in the stone field built without error")
		}
	}()
	buildRogueWithConsumes(t, &proto.Consumes{
		MainHandImbue: proto.WeaponImbue_DeadlyPoison,
		MainHandStone: proto.WeaponImbue_InstantPoison,
	})
}
