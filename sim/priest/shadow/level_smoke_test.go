package shadow

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test; see
// sim/hunter/level_smoke_test.go for the full explanation.
func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "ShadowPriest",
		Class:       proto.Class_ClassPriest,
		Race:        proto.Race_RaceUndead,
		Talents:     P1Talents,
		SpecOptions: PlayerOptionsBasic,
	})
}
