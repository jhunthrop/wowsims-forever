package dpswarrior

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test; see
// sim/hunter/level_smoke_test.go for the full explanation. This is the
// regression test for the warrior half of the defect the smoke was
// written to catch: Slam and five other per-rank cooldown abilities
// (Bloodthirst, Mortal Strike, Overpower, Pummel, Thunder Clap; also
// Shield Slam in sim/warrior/tank_warrior) set a Cast.CD with a Timer
// even at a rank whose generated cooldown was 0 - which panics
// ("Cast.CD w/o Duration") the moment such a rank is reached, e.g. Slam
// rank 2 at level 38 - fixed in the six files under sim/warrior/.
func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "FuryWarrior",
		Class:       proto.Class_ClassWarrior,
		Race:        proto.Race_RaceOrc,
		Talents:     P1Talents,
		SpecOptions: PlayerOptionsFury,
	})
}
