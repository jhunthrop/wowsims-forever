package hunter

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// A hunter pet's level follows its owner's level exactly (core.NewPet
// sets it from owner.Level; pet.go's own comment cites this) - the level-
// aware sim design's third named test, "a hunter's pet is the owner's
// level." Built at level 30, not CharacterMaxLevel, so a hard-coded
// CharacterMaxLevel default couldn't pass this by accident.
func TestHunterPetIsOwnerLevel(t *testing.T) {
	const wantLevel = 30

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              wantLevel,
			Equipment:          &proto.EquipmentSpec{}, // no gear: this fork's item database isn't generated in this test environment (sim/core/items/all_items.go, a `make items` output), and this test only needs the pet's level, not its gear-derived stats.
			Buffs:              core.FullBuffs.Player,
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)

	built, ok := env.Raid.Parties[0].Players[0].(*Hunter)
	if !ok {
		t.Fatal("player 0 did not build as a *Hunter")
	}
	if built.Level != wantLevel {
		t.Fatalf("hunter.Level = %d, want %d", built.Level, wantLevel)
	}
	if built.pet == nil {
		t.Fatal("hunter has no pet; P1PlayerOptions sets PetType Cat and PetUptime 1")
	}
	if built.pet.Level != wantLevel {
		t.Errorf("hunter pet.Level = %d, want the owner's level %d", built.pet.Level, wantLevel)
	}
}
