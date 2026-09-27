package dps

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test; see
// sim/hunter/level_smoke_test.go for the full explanation. This package
// carries two talent presets (SM/Ruin and DS/Ruin), so both are
// exercised.
func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "SMRuinWarlock",
		Class:       proto.Class_ClassWarlock,
		Race:        proto.Race_RaceOrc,
		Talents:     TalentsSMRuin,
		SpecOptions: DefaultDestroWarlock,
	})
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "DSRuinWarlock",
		Class:       proto.Class_ClassWarlock,
		Race:        proto.Race_RaceOrc,
		Talents:     TalentsDSRuin,
		SpecOptions: DefaultDestroWarlock,
	})
}
