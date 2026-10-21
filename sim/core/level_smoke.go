package core

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// LevelSmokeLevels are the levels every class/spec preset's smoke test
// builds a character and runs a short sim at (level-aware-sim-design.md:
// "build a character at levels 1, 10, 20, 30, 40, 50 and 60"). 38 is
// added to that list: it is not one of Season of Discovery's 25/40/50/60
// talent brackets, and a shaman-elemental and a warrior build each
// panicked only at a level like it - a rank a bracket-keyed table or a
// per-rank cooldown array had never been asked for before.
var LevelSmokeLevels = []int32{1, 10, 20, 30, 38, 40, 50, 60}

// LevelSmokePreset is one class/spec combination a level_smoke_test.go
// in a class package exercises: the same Class/Race/Talents/SpecOptions
// a P1-style regression test in that package already builds, minus the
// gear and minus the exact rotation ranks (RunLevelSmoke supplies both:
// no gear, since this fork's item database isn't generated in this test
// environment, and Rotation is optional because a raw, un-rank-rewritten
// APL would just cast ids the level hasn't learned - see the doc
// comment on Rotation).
type LevelSmokePreset struct {
	Label       string
	Class       proto.Class
	Race        proto.Race
	Talents     string
	SpecOptions interface{}

	// Rotation is optional. sim/request rewrites an embedded APL's
	// max-rank spellIds to the level's learned ranks before handing it
	// to this engine (level-aware-sim-design.md point 4); that rewrite
	// lives in a different repo, not this fork, so a level_smoke_test.go
	// that wants a rotation exercised has to reproduce it by hand (see
	// sim/shaman/elemental/level_smoke_test.go). Leaving Rotation nil
	// still exercises every spell's registration - where the two defects
	// this smoke was written to catch actually panicked - just not the
	// AI choosing to cast one.
	Rotation *proto.APLRotation

	// DistanceFromTarget defaults to 5 (melee range) when zero; a caster
	// or ranged preset that needs to be out of melee range sets it.
	DistanceFromTarget float64
}

// smokeEncounter is the short, single-target, single-iteration encounter
// every level's build runs a sim against - long enough for a GCD-bound
// rotation to cast several times, short enough that sixty-odd of these
// across every class stay fast.
func smokeEncounter() *proto.Encounter {
	return &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{NewDefaultTarget()},
	}
}

// RunLevelSmoke builds preset at every LevelSmokeLevels level with no
// gear and runs a short sim, so the level's own subtest can fail on any
// panic or sim error without stopping the other levels' subtests (each
// gets its own recover) - the level-aware sim design's smoke test,
// factored once here so no class package repeats the loop, the encounter,
// or the panic isolation (AGENTS.md: no code duplication across
// features). It also asserts the built character's Level equals the
// level requested, so a hard-coded CharacterMaxLevel default couldn't
// pass this by accident (sim/hunter/pet_level_test.go's same check).
func RunLevelSmoke(t *testing.T, preset LevelSmokePreset) {
	t.Helper()

	distance := preset.DistanceFromTarget
	if distance == 0 {
		distance = 5
	}

	for _, level := range LevelSmokeLevels {
		level := level
		t.Run(fmt.Sprintf("%s/L%d", preset.Label, level), func(t *testing.T) {
			t.Helper()
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic at level %d: %v\n%s", level, r, debug.Stack())
				}
			}()

			player := WithSpec(&proto.Player{
				Class:              preset.Class,
				Race:               preset.Race,
				Level:              level,
				Equipment:          &proto.EquipmentSpec{},
				Buffs:              FullBuffs.Player,
				TalentsString:      preset.Talents,
				Rotation:           preset.Rotation,
				DistanceFromTarget: distance,
			}, preset.SpecOptions)

			raid := SinglePlayerRaidProto(player, FullBuffs.Party, FullBuffs.Raid, FullBuffs.Debuffs)

			env, _, _ := NewEnvironment(raid, smokeEncounter(), true)
			built := env.Raid.Parties[0].Players[0].GetCharacter()
			if built.Level != level {
				t.Fatalf("built Level = %d, want %d", built.Level, level)
			}

			result := RunRaidSim(&proto.RaidSimRequest{
				Raid:      raid,
				Encounter: smokeEncounter(),
				SimOptions: &proto.SimOptions{
					Iterations: 1,
					RandomSeed: 1,
					IsTest:     true,
				},
			})
			if result.Error != nil {
				t.Fatalf("sim error at level %d: %s", level, result.Error.Message)
			}
		})
	}
}
