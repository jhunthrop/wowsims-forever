package core

import (
	"testing"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/classic/sim/core/proto"
)

// Forever's biome-conditional trinkets need the encounter to know where it
// is. The concept is additive: an encounter that does not set one reads
// BiomeUnknown, which matches nothing, which is a vanilla fight.
func TestEncounterCarriesItsBiome(t *testing.T) {
	enc := NewEncounter(&proto.Encounter{
		Duration: 180,
		Biome:    proto.Biome_BiomeVolcanic,
		Targets:  []*proto.Target{DefaultTargetProtoLvl60},
	})
	if enc.Biome != proto.Biome_BiomeVolcanic {
		t.Errorf("Encounter.Biome = %v, want BiomeVolcanic", enc.Biome)
	}
}

func TestEncounterWithoutABiomeIsUnknown(t *testing.T) {
	enc := NewEncounter(&proto.Encounter{
		Duration: 180,
		Targets:  []*proto.Target{DefaultTargetProtoLvl60},
	})
	if enc.Biome != proto.Biome_BiomeUnknown {
		t.Errorf("Encounter.Biome = %v, want BiomeUnknown for an encounter that sets none", enc.Biome)
	}
}

// An item effect is written against a unit, so a unit must be able to ask
// where it is without the effect plumbing the environment itself.
func TestUnitReadsTheEncounterBiome(t *testing.T) {
	env := setupBiomeEnv(t, proto.Biome_BiomeSwamp, proto.MobType_MobTypeBeast)
	player := env.Raid.Parties[0].Players[0].GetCharacter().Unit
	if got := player.Biome(); got != proto.Biome_BiomeSwamp {
		t.Errorf("Unit.Biome() = %v, want BiomeSwamp", got)
	}
	if got := env.Encounter.TargetUnits[0].MobType; got != proto.MobType_MobTypeBeast {
		t.Errorf("target MobType = %v, want MobTypeBeast", got)
	}
}

// A unit with no environment yet (during registration, before the
// environment is constructed) must answer BiomeUnknown rather than panic.
func TestUnitBiomeBeforeEnvironmentIsUnknown(t *testing.T) {
	u := &Unit{Type: PlayerUnit}
	if got := u.Biome(); got != proto.Biome_BiomeUnknown {
		t.Errorf("Unit.Biome() with no environment = %v, want BiomeUnknown", got)
	}
}

// setupBiomeEnv builds the smallest environment that has a player and one
// target, so the biome plumbing can be read end to end.
func setupBiomeEnv(t *testing.T, biome proto.Biome, mob proto.MobType) *Environment {
	t.Helper()
	target := googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
	target.MobType = mob
	env, _, _ := NewEnvironment(
		SinglePlayerRaidProto(&proto.Player{
			Name:      "Biome Test",
			Race:      proto.Race_RaceOrc,
			Class:     proto.Class_ClassShaman,
			Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment: &proto.EquipmentSpec{},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		&proto.Encounter{Duration: 180, Biome: biome, Targets: []*proto.Target{target}},
		false,
	)
	return env
}
