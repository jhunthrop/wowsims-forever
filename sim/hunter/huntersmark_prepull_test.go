package hunter

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Hunter's Mark is a debuff a hunter puts on the boss before the pull. It
// deals no damage, so the engine's hold that times a damaging pre-pull cast
// to land at 0 must leave it alone: held to the pull, it spent the opening
// global and pushed every opener cast back 1.5 s (sim-accuracy round 21,
// Survival -0.4% at raid 60 once its card cast the mark before the pull).
func TestPrepullHuntersMarkIsCastBeforeThePull(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			DistanceFromTarget: 25,
			Rotation:           prepullOnlyRotation(14325, "-6s"),
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 30, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true, Debug: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %v", result.Error.Message)
	}
	for _, line := range strings.Split(result.Logs, "\n") {
		if strings.Contains(line, "Casting {SpellID: 14325}") {
			if !strings.HasPrefix(line, "[-6.00]") {
				t.Errorf("pre-pull Hunter's Mark cast at %q, want at -6.00 s: a debuff with no damage is not held to the pull", line)
			}
			return
		}
	}
	t.Fatal("the pre-pull Hunter's Mark was never cast")
}
