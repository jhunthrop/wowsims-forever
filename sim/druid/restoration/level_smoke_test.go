package restoration

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test; see
// sim/hunter/level_smoke_test.go for the full explanation.
func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "RestorationDruid",
		Class:       proto.Class_ClassDruid,
		Race:        proto.Race_RaceTauren,
		Talents:     StandardTalents,
		SpecOptions: PlayerOptionsStandard,
	})
}
