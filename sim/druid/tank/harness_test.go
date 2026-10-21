package tank

import (
	"os"
	"testing"

	googleproto "google.golang.org/protobuf/proto"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

func init() {
	RegisterFeralTankDruid()
}

// Tank harness: the boss profile the site's ranker publishes. Level 63, no
// creature type, one melee swing every 2 s rolled 2400 to 3192 before armor
// (the +3 level gap gives crushing blows), the druid tanking it and in front
// of it, and a healer that keeps the druid topped up in bursts.
const (
	harnessBossLevel     = 63
	harnessBossSwing     = 2.0
	harnessBossMinDamage = 2400.0
	harnessBossSpread    = 0.33
	harnessDuration      = 180.0
	harnessIterations    = 300
	harnessHealerHps     = 600.0
	harnessHealCadence   = 2.0
	harnessHealVariation = 0.2
	harnessBurstWindow   = 6
)

func harnessBoss() *proto.Target {
	return &proto.Target{
		Level:         harnessBossLevel,
		MinBaseDamage: harnessBossMinDamage,
		DamageSpread:  harnessBossSpread,
		SwingSpeed:    harnessBossSwing,
		TankIndex:     0,
	}
}

func bearPlayer(level int32, talents string, gear *proto.EquipmentSpec, rotation *proto.APLRotation) *proto.Player {
	return core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassDruid,
		Race:               proto.Race_RaceTauren,
		Level:              level,
		Equipment:          gear,
		Consumes:           &proto.Consumes{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talents,
		Rotation:           rotation,
		InFrontOfTarget:    true,
		DistanceFromTarget: 5,
		HealingModel: &proto.HealingModel{
			Hps:              harnessHealerHps,
			CadenceSeconds:   harnessHealCadence,
			CadenceVariation: harnessHealVariation,
			BurstWindow:      harnessBurstWindow,
		},
	}, defaultBearOptions())
}

func defaultBearOptions() *proto.Player_FeralTankDruid {
	return &proto.Player_FeralTankDruid{
		FeralTankDruid: &proto.FeralTankDruid{
			Options: &proto.FeralTankDruid_Options{InnervateTarget: &proto.UnitReference{}},
		},
	}
}

// harnessDebuffs are the full raid debuffs minus Demoralizing Roar, which is
// the bear's own job.
func harnessDebuffs() *proto.Debuffs {
	debuffs := googleproto.Clone(core.FullBuffs.Debuffs).(*proto.Debuffs)
	debuffs.DemoralizingRoar = proto.TristateEffect_TristateEffectMissing
	return debuffs
}

func bearRaid(player *proto.Player) *proto.Raid {
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, harnessDebuffs())
	raid.Tanks = append(raid.Tanks, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0})
	return raid
}

func harnessEncounter() *proto.Encounter {
	return &proto.Encounter{Duration: harnessDuration, Targets: []*proto.Target{harnessBoss()}}
}

func loadGear(t *testing.T) *proto.EquipmentSpec {
	t.Helper()
	return core.GetGearSet("../../../ui/feral_tank_druid/gear_sets", "forever_l60").GearSet
}

func loadRotation(t *testing.T, path string) *proto.APLRotation {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the rotation %s: %v", path, err)
	}
	return core.APLRotationFromJsonString(string(data))
}

// runHarness runs the tank fight and returns the druid's metrics.
func runHarness(t *testing.T, player *proto.Player, iterations int32) *proto.UnitMetrics {
	t.Helper()
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:      bearRaid(player),
		Encounter: harnessEncounter(),
		SimOptions: &proto.SimOptions{
			Iterations: iterations,
			RandomSeed: 101,
			IsTest:     true,
		},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	return result.RaidMetrics.Parties[0].Players[0]
}

var _ = druid.Bear
