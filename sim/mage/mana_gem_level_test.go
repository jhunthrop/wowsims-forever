package mage

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// A mage conjures each gem at the level the trainer teaches it: Conjure Mana
// Agate at 28 (trainables 759), Jade at 38 (3552), Citrine at 48 (10053) and
// Ruby at 58 (10054). The engine read the Mana Agate item's required level
// (5514, 23) instead, so a mage at 23 to 27 popped a gem the class cannot
// make yet; the rank rows judged on the leveling walk at 24 to 27 saw that
// mana.
func TestManaGemsRegisterAtTheirConjureLevels(t *testing.T) {
	for _, tc := range []struct {
		level int32
		ranks []int
	}{
		{23, nil}, {27, nil}, {28, []int{1}}, {37, []int{1}}, {38, []int{1, 2}},
		{47, []int{1, 2}}, {48, []int{1, 2, 3}}, {57, []int{1, 2, 3}}, {58, []int{1, 2, 3, 4}},
	} {
		player := core.WithSpec(&proto.Player{
			Class: proto.Class_ClassMage, Race: proto.Race_RaceTroll, Level: tc.level,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              &proto.IndividualBuffs{},
			DistanceFromTarget: 20,
			Rotation:           &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}, PlayerOptions)
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
			SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
		}, simsignals.CreateSignals())
		mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
		var got []int
		for rank := 1; rank <= ManaGemRanks; rank++ {
			if mage.ManaGem[rank] != nil {
				got = append(got, rank)
			}
		}
		if len(got) != len(tc.ranks) || func() bool {
			for i := range got {
				if got[i] != tc.ranks[i] {
					return true
				}
			}
			return false
		}() {
			t.Errorf("level %d: mana gem ranks registered = %v, want %v", tc.level, got, tc.ranks)
		}
	}
}
