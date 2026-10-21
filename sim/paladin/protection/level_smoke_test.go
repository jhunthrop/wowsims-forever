package protection

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test; see
// sim/hunter/level_smoke_test.go for the full explanation.
func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "ProtectionPaladin",
		Class:       proto.Class_ClassPaladin,
		Race:        proto.Race_RaceHuman,
		Talents:     paladin.ForeverProtectionTalents,
		SpecOptions: PlayerOptionsRighteousFury,
	})
}
