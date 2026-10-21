package enhancement

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test; see
// sim/hunter/level_smoke_test.go for the full explanation.
func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "EnhancementShaman",
		Class:       proto.Class_ClassShaman,
		Race:        proto.Race_RaceTroll,
		Talents:     DefaultTalents,
		SpecOptions: PlayerOptionsSyncAuto,
	})
}
