package restoration

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get caster sets included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
	"github.com/wowsims/classic/sim/healsim"
)

func init() {
	RegisterRestorationDruid()
}

// StandardTalents is the build the conformance preset and these tests use: the
// healing half of the Restoration tree (40 points: Nature's Focus 5, Naturalist
// 5, Reflection 3, Gift of Nature 5, Gift of the Earthmother, Tranquil Spirit 5,
// Improved Rejuvenation 3, Swiftmend, Nature's Swiftness, Living Spirit 3,
// Improved Tranquility 2, Improved Regrowth 5, Wild Growth) and 11 Balance
// points for Genesis 5, Nature's Majesty 2, Moonglow 3 and Nature's Splendor.
const StandardTalents = "05302001--5050035153113251"

// healerBonusStats is what a geared level 60 healer adds to the bare class:
// the sims here wear no items, so the stats come in as bonus stats.
var healerBonusStats = stats.Stats{
	stats.Intellect:    250,
	stats.Spirit:       180,
	stats.HealingPower: 700,
	stats.MP5:          20,
}

var PlayerOptionsStandard = &proto.Player_RestorationDruid{
	RestorationDruid: &proto.RestorationDruid{
		Options: &proto.RestorationDruid_Options{
			InnervateTarget: &proto.UnitReference{Type: proto.UnitReference_Self},
		},
	},
}

// newPlayer is a restoration druid of the given level with the given talents
// and bonus stats, running the given rotation (nil for an empty one).
func newPlayer(level int32, talents string, bonus stats.Stats, rotation *proto.APLRotation) *proto.Player {
	if rotation == nil {
		rotation = &proto.APLRotation{}
	}
	return core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassDruid,
		Race:          proto.Race_RaceTauren,
		Level:         level,
		Equipment:     &proto.EquipmentSpec{},
		TalentsString: talents,
		BonusStats:    &proto.UnitStats{Stats: bonus.ToFloatArray()},
		Rotation:      rotation,
	}, PlayerOptionsStandard)
}

// newDruid builds a sim around the player, reset and ready, and returns the
// druid in it. The raid is the healer alone: heals land on the druid itself.
func newDruid(t *testing.T, player *proto.Player) (*RestorationDruid, *core.Simulation) {
	t.Helper()
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	resto, ok := sim.Raid.Parties[0].Players[0].(*RestorationDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *RestorationDruid")
	}
	return resto, sim
}

// talentsString builds a talent string from node ranks by proto field name.
func talentsString(t *testing.T, ranks map[string]int) string {
	t.Helper()
	built, err := core.TalentsStringFromRanks((&proto.DruidTalents{}).ProtoReflect(), druid.TalentTreeSizes, ranks)
	if err != nil {
		t.Fatal(err)
	}
	return built
}

// runHealing runs the player against the shared test profile.
func runHealing(t *testing.T, player *proto.Player, duration float64, iterations int32) healsim.Summary {
	t.Helper()
	summary, err := healsim.Run(healsim.Request(player, healsim.TestProfile(), duration, iterations))
	if err != nil {
		t.Fatal(err)
	}
	return summary
}
