package hunter

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test
// (docs/superpowers/specs/2026-09-27-level-aware-sim-design.md): it
// builds this package's P1 preset at every level in
// core.LevelSmokeLevels, with no gear, and runs a short sim so the
// class's registration code and its rotation both run at levels other
// than 60.
func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:              "Hunter",
		Class:              proto.Class_ClassHunter,
		Race:               proto.Race_RaceOrc,
		Talents:            P1Talents,
		SpecOptions:        P1PlayerOptions,
		DistanceFromTarget: 25, // ranged: outside core.MaxMeleeAttackDistance so shots and pet melee both behave.
	})
}
