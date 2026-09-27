package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// newTestCharacter builds a Character directly through NewCharacter - not
// through the full NewAgent/agent-factory path, which needs a class
// package's RegisterX() to have run - so this stays usable from sim/core
// itself without an import cycle. A non-nil oneof Spec value is required:
// PlayerProtoToSpec (agent.go) reflects on it, and a nil oneof panics
// there regardless of whether any agent factory is registered for it.
func newTestCharacter(t *testing.T, level int32, class proto.Class, race proto.Race) Character {
	t.Helper()
	player := &proto.Player{
		Race:      race,
		Class:     class,
		Level:     level,
		Equipment: &proto.EquipmentSpec{},
		Spec:      &proto.Player_Warrior{Warrior: &proto.Warrior{}},
	}
	return NewCharacter(&Party{}, 0, player)
}

// A character built at a level in range 1..CharacterMaxLevel keeps that
// exact level, and its base stats come from that level's row in the
// generated table plus the level's Attack Power offset.
func TestNewCharacterHonorsPlayerLevel(t *testing.T) {
	character := newTestCharacter(t, 30, proto.Class_ClassWarrior, proto.Race_RaceHuman)
	if character.Level != 30 {
		t.Fatalf("character.Level = %d, want 30", character.Level)
	}
	if got, want := character.GetStat(stats.Agility), 44.0; got != want {
		t.Errorf("level-30 Warrior Agility = %v, want %v (wowhead's table)", got, want)
	}
	if got, want := character.GetStat(stats.AttackPower), 70.0; got != want {
		t.Errorf("level-30 Warrior base Attack Power = %v, want %v (3*30-20)", got, want)
	}
}

// A level-0 proto.Player - the field's zero value, meaning every request
// that predates it, and every hand-built test fixture that never sets it
// - must still build a character at CharacterMaxLevel with today's exact
// numbers: this is the golden-equality regression design section 2 names
// ("a level-0 player builds at 60 unchanged").
func TestNewCharacterZeroLevelBuildsAtCharacterMaxLevelUnchanged(t *testing.T) {
	for _, class := range allClasses {
		zero := newTestCharacter(t, 0, class, proto.Race_RaceHuman)
		if zero.Level != CharacterMaxLevel {
			t.Fatalf("%v: character.Level = %d, want CharacterMaxLevel (%d)", class, zero.Level, CharacterMaxLevel)
		}
		max := newTestCharacter(t, CharacterMaxLevel, class, proto.Race_RaceHuman)
		if got, want := zero.GetStats(), max.GetStats(); got != want {
			t.Errorf("%v level-0 base stats = %+v, want the explicit CharacterMaxLevel build's %+v", class, got, want)
		}
		// And that shared value is today's numbers: ClassBaseStats plus
		// ClassBaseCrit plus the universal dependencies every character
		// gets (addUniversalStatDependencies), with RaceHuman's all-zero
		// offset contributing nothing.
		wantBase := ClassBaseStats[class].Add(ClassBaseCrit[class])
		wantBase[stats.Health] += 20 - 10*20
		wantBase[stats.Parry] += 5 * ParryRatingPerParryChance
		wantBase[stats.Block] += 5 * BlockRatingPerBlockChance
		if got := max.GetStats(); got != wantBase {
			t.Errorf("%v CharacterMaxLevel base stats = %+v, want %+v", class, got, wantBase)
		}
	}
}

// An out-of-range level (above CharacterMaxLevel) must not build a
// character at a level the generated tables have no row for; it falls
// back to CharacterMaxLevel exactly like the zero-value case.
func TestNewCharacterAboveMaxLevelBuildsAtCharacterMaxLevel(t *testing.T) {
	character := newTestCharacter(t, CharacterMaxLevel+30, proto.Class_ClassMage, proto.Race_RaceHuman)
	if character.Level != CharacterMaxLevel {
		t.Fatalf("character.Level = %d, want CharacterMaxLevel (%d)", character.Level, CharacterMaxLevel)
	}
}
