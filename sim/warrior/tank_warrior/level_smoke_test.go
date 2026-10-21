package tankwarrior

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test; see
// sim/hunter/level_smoke_test.go for the full explanation. Protection's
// Shield Slam shares Slam's defect (sim/warrior/dps_warrior's
// TestLevelSmoke doc comment): rank 0 of its generated cooldown table -
// what a warrior below level 40 with the talent already spent resolves
// to - is 0, and a Cast.CD with a Timer but no Duration panics. Fixed in
// sim/warrior/shield_slam.go.
func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "ProtectionWarrior",
		Class:       proto.Class_ClassWarrior,
		Race:        proto.Race_RaceOrc,
		Talents:     warrior.ForeverProtectionTalents,
		SpecOptions: PlayerOptionsBasic,
	})
}
