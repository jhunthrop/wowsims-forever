package hunter

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// A request may hand a rage potion to a class with no rage bar: the API
// takes any consumable id, and the /sim page's preset is one list for every
// physical spec. The potion's rage has nowhere to go, but the run must not
// panic on the missing bar (it used to dereference nil in AddRage).
func TestARagePotionOnAClassWithNoRageBarRunsCleanly(t *testing.T) {
	autocast := &proto.APLListItem{Action: &proto.APLAction{Action: &proto.APLAction_AutocastOtherCooldowns{
		AutocastOtherCooldowns: &proto.APLActionAutocastOtherCooldowns{},
	}}}
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Consumes:           &proto.Consumes{DefaultPotion: proto.Potions_MightyRagePotion},
			DistanceFromTarget: 25,
			Rotation:           &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{autocast}},
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 30, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{Iterations: 5, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %v", result.Error.Message)
	}
}
