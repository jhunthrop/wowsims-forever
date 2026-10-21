package rogue_test

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects and sets included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
)

// The Twin Blades of Hakkari (client ItemSet 461, build 1.60.1.70009):
// the Warblade of the Hakkari pair, items 19865 and 19866. The set's
// single bonus is spell 15763 at two pieces, "Increased Swords +6": a
// skill aura (aura 30) on skill line 43, one-handed Swords, base points
// 6. Zul'Gurub is not in the Phase 1 raid list (research/01-official-facts.md:
// Barrow Deeps, Hyjal Summit and Onyxia open on 9 December), so the
// bonus has no Phase 1 effect; it is modelled because the client states
// it and the items are in the item table.
func swordsSkillWearing(t *testing.T, mainHand, offHand int32) float64 {
	t.Helper()
	if !core.WITH_DB {
		t.Skip("needs the item database (--tags=with_db)")
	}
	items := make([]*proto.ItemSpec, 17)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: mainHand}
	items[proto.ItemSlot_ItemSlotOffHand] = &proto.ItemSpec{Id: offHand}

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassRogue,
			Race:               proto.Race_RaceHuman,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{Items: items},
			Buffs:              &proto.IndividualBuffs{},
			DistanceFromTarget: 5,
		},
		&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	built, ok := env.Raid.Parties[0].Players[0].(*dpsrogue.DpsRogue)
	if !ok {
		t.Fatal("player 0 did not build as a *dpsrogue.DpsRogue")
	}
	return built.PseudoStats.SwordsSkill
}

const (
	warbladeOfTheHakkariMainHand int32 = 19865
	warbladeOfTheHakkariOffHand  int32 = 19866
)

func TestTwinBladesOfHakkariAddsSixSwordsSkillAtTwoPieces(t *testing.T) {
	oneBlade := swordsSkillWearing(t, warbladeOfTheHakkariMainHand, 0)
	bothBlades := swordsSkillWearing(t, warbladeOfTheHakkariMainHand, warbladeOfTheHakkariOffHand)
	if got := bothBlades - oneBlade; got != 6 {
		t.Errorf("the pair adds %v Swords skill over one blade, want 6", got)
	}
}
