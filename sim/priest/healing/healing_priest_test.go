package healing

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get caster sets included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
	"github.com/wowsims/classic/sim/priest"
)

func init() {
	RegisterHealingPriest()
}

// The two reference builds, each a valid level-60 spend of the live trees
// (51 points; tier and prerequisite rules checked against
// data/builds/1.60.1.70009/talents/priest.json). Holy puts 35 points in
// the Holy tree and 16 in Discipline (Twin Disciplines, Improved Power
// Word: Shield, Silent Resolve, Mental Agility, Inner Focus, Meditation);
// Discipline puts 33 in Discipline (down to Penance, Renewed Hope, Divine
// Aegis and Power Infusion) and 18 in Holy (Improved Renew, Holy
// Specialization, Divine Fury, Inspiration, Improved Healing). The
// conformance preset (sim/conformance/presets.go) carries HolyTalents.
const (
	HolyTalents = "0052030312-33505003030121531"
	DiscTalents = "005203031305101531-0350500302"
)

// raidHealerStats stands in for gear, which the engine's item database does
// not supply to tests: roughly a tier-1 raid healer.
var raidHealerStats = stats.Stats{
	stats.Intellect:    300,
	stats.Spirit:       200,
	stats.HealingPower: 700,
	stats.MP5:          40,
}

// healer builds the healing priest every test drives.
func healer(level int32, talents string, options *proto.HealingPriest_Options, rotation *proto.APLRotation) *proto.Player {
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          proto.Race_RaceHuman,
		Level:         level,
		Equipment:     &proto.EquipmentSpec{},
		BonusStats:    &proto.UnitStats{Stats: raidHealerStats.ToFloatArray()},
		TalentsString: talents,
		Rotation:      rotation,
	}, &proto.Player_HealingPriest{HealingPriest: &proto.HealingPriest{Options: options}})
	return player
}

// healerSim is a prepulled sim with the healer next to the fake raid and
// no rotation, so a test drives the spells directly.
func healerSim(t *testing.T, level int32, talents string) (*core.Simulation, *HealingPriest) {
	t.Helper()
	return healerSimWith(t, level, talents, &proto.HealingPriest_Options{})
}

// healerSimWith is healerSim with the spec's options set.
func healerSimWith(t *testing.T, level int32, talents string, options *proto.HealingPriest_Options) (*core.Simulation, *HealingPriest) {
	t.Helper()
	return agentSim(t, healer(level, talents, options, &proto.APLRotation{}))
}

// agentSim prepulls a sim around player and returns the healing priest.
func agentSim(t *testing.T, player *proto.Player) (*core.Simulation, *HealingPriest) {
	t.Helper()
	req := healsim.Request(player, healsim.TestProfile(), 300, 1)
	sim := core.NewSim(req, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	agent, ok := sim.Raid.Parties[0].Players[0].(*HealingPriest)
	if !ok {
		t.Fatal("the raid's first player is not a healing priest")
	}
	return sim, agent
}

var _ priest.PriestAgent = (*HealingPriest)(nil)
