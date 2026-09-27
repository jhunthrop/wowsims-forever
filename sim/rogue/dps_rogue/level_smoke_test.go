package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestLevelSmoke is the level-aware sim design's engine smoke test; see
// sim/hunter/level_smoke_test.go for the full explanation. This package
// carries two talent presets (Combat Swords and Combat Daggers), so both
// are exercised.
//
// Skipped like this package's own TestCombatSinisterStrike/
// TestCombatDaggers: rogue.applyWeaponExpertise (sim/rogue/talents.go:288)
// indexes a 3-entry []float64{0, 3, 5} table with rogue.Talents.
// WeaponExpertise, a talent whose Forever trait tree - not yet rewritten
// for this spec - goes up to 5 points; both presets spend all 5, so
// building either panics with "index out of range [5] with length 3" at
// EVERY level, level 60 included, before any spell gets registered. That
// makes it the Forever-talent-rewrite gap the skip already names, not a
// level-awareness defect - guessing bonus values for points 3-5 would be
// inventing game data this smoke has no source for.
func TestLevelSmoke(t *testing.T) {
	core.SkipAwaitingForeverTalentRewrite(t, "sim/rogue/dps_rogue")
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "CombatSwordsRogue",
		Class:       proto.Class_ClassRogue,
		Race:        proto.Race_RaceHuman,
		Talents:     CombatSwordsTalents,
		SpecOptions: DefaultRogue,
	})
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:       "CombatDaggersRogue",
		Class:       proto.Class_ClassRogue,
		Race:        proto.Race_RaceHuman,
		Talents:     CombatDaggersTalents,
		SpecOptions: DefaultRogue,
	})
}
