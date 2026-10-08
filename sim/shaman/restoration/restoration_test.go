package restoration

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
	"github.com/wowsims/classic/sim/shaman"
)

func init() {
	RegisterRestorationShaman()
}

// FullTalents is the whole Restoration tree, 51 points: Improved Healing
// Wave to Riptide in the client's order.
const FullTalents = "--5533523315513151"

// Positions of the Restoration talents in the client's order, for
// restoTalents.
const (
	posImprovedHealingWave = 1 + iota
	posTotemicFocus
	posMindfulness
	posNaturalGrace
	posTidalFocus
	posImprovedReincarnation
	posAncestralHealing
	posHealingFocus
	posWaterShield
	posTidalMastery
	posRestorativeTotems
	posManaTideTotem
	posHealingWay
	posNaturesSwiftness
	posPurification
	posRiptide
	restorationTreeSize = posRiptide
)

// restoTalents is a talent string with only the given Restoration talents
// (position to ranks) taken, so a test sees one talent's effect and nothing
// else's.
func restoTalents(ranks map[int]int) string {
	digits := make([]byte, restorationTreeSize)
	for i := range digits {
		digits[i] = '0'
	}
	for position, rank := range ranks {
		digits[position-1] = byte('0' + rank)
	}
	return "--" + string(digits)
}

// riptideOnly takes Riptide and nothing else.
var riptideOnly = restoTalents(map[int]int{posRiptide: 1})

// testHealingPower is the bonus healing every numeric test gives the healer,
// so a coefficient shows up in the heal.
const testHealingPower = 400

var restorationOptions = &proto.Player_RestorationShaman{
	RestorationShaman: &proto.RestorationShaman{Options: &proto.RestorationShaman_Options{}},
}

// newHealerPlayer is a gearless Troll Restoration shaman with a fixed amount
// of bonus stats, so the only healing power, mana and crit in a test are the
// ones it states.
func newHealerPlayer(level int32, talents string, rotation *proto.APLRotation) *proto.Player {
	bonus := stats.Stats{
		stats.Intellect:    300,
		stats.Spirit:       120,
		stats.HealingPower: testHealingPower,
		stats.MP5:          30,
	}
	return core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassShaman,
		Race:          proto.Race_RaceTroll,
		Level:         level,
		Equipment:     &proto.EquipmentSpec{},
		TalentsString: talents,
		BonusStats:    &proto.UnitStats{Stats: bonus.ToFloatArray()},
		Rotation:      rotation,
	}, restorationOptions)
}

// quietRaid is a fake raid that is healthy and takes no damage, so a test
// controls every point of health itself.
func quietRaid() *proto.RaidDamageModel {
	return &proto.RaidDamageModel{Profile: "quiet", TankHealth: 9000, MemberHealth: 5000}
}

// newHealer builds the healer in the fake raid, reset and ready to cast.
func newHealer(t *testing.T, level int32, talents string) (*core.Simulation, *RestorationShaman) {
	t.Helper()

	req := healsim.Request(newHealerPlayer(level, talents, nil), quietRaid(), 60, 1)
	sim := core.NewSim(req, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	healer, ok := sim.Raid.Parties[0].Players[healsim.HealerIndex].(*RestorationShaman)
	if !ok {
		t.Fatal("the raid's first player is not a Restoration shaman")
	}
	return sim, healer
}

// raidMember is the fake raid member at a raid index.
func raidMember(sim *core.Simulation, index int32) *core.Unit {
	return sim.Raid.AllPlayerUnits[index]
}

func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "RestorationShaman",
		Class:       proto.Class_ClassShaman,
		Race:        proto.Race_RaceTroll,
		Talents:     FullTalents,
		SpecOptions: restorationOptions,
	})
}

func TestTheTankIsTheMainTarget(t *testing.T) {
	sim, healer := newHealer(t, 60, "")

	if got, want := healer.GetMainTarget(), raidMember(sim, healsim.TankIndex); got != want {
		t.Errorf("main target = %s, want the tank (raid member %d)", got.Label, healsim.TankIndex)
	}
}

func TestWithoutAFakeRaidTheShamanIsItsOwnTarget(t *testing.T) {
	raid := core.SinglePlayerRaidProto(newHealerPlayer(60, "", nil), &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{Duration: 10, Targets: []*proto.Target{core.NewDefaultTarget()}}, true)

	healer := env.Raid.Parties[0].Players[0].(*RestorationShaman)
	if healer.GetMainTarget() != &healer.Unit {
		t.Error("a raid of one should make the healer its own main target")
	}
}

func TestShamanAgentExposesTheShaman(t *testing.T) {
	_, healer := newHealer(t, 60, "")

	var agent shaman.ShamanAgent = healer
	if agent.GetShaman() != healer.Shaman {
		t.Error("GetShaman should return the embedded shaman")
	}
}
