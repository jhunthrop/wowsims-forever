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

// NewBiomeDamageEffect is the one function the whole biome feature
// exists for, and until this test it had no caller and no coverage: the
// plumbing above was asserted and the consumer was not. It also reads
// the biome eagerly, inside the item effect, where its sibling
// NewMobTypeDamageEffect defers to RegisterPostFinalizeEffect — so what
// this pins is that the eager read is safe, i.e. that
// Environment.construct sets the Encounter (and each unit's Env) before
// initialize() runs the item effects. If that order ever changes, this
// fails instead of the trinket silently doing nothing.
func TestBiomeDamageEffectAppliesOnlyInItsBiome(t *testing.T) {
	item := trinketWithoutAnEffect(t)
	const multiplier = 1.1
	NewBiomeDamageEffect(item, []proto.Biome{proto.Biome_BiomeVolcanic}, multiplier)

	for _, tc := range []struct {
		biome proto.Biome
		want  float64
	}{
		{proto.Biome_BiomeVolcanic, multiplier},
		{proto.Biome_BiomeSwamp, 1},
		{proto.Biome_BiomeUnknown, 1},
	} {
		t.Run(tc.biome.String(), func(t *testing.T) {
			with := biomeEnvWearing(t, tc.biome, item).Raid.Parties[0].Players[0].GetCharacter()
			without := biomeEnvWearing(t, tc.biome, 0).Raid.Parties[0].Players[0].GetCharacter()
			got := with.PseudoStats.DamageDealtMultiplier / without.PseudoStats.DamageDealtMultiplier
			if got != tc.want {
				t.Errorf("in %v the trinket's damage multiplier is %v, want %v", tc.biome, got, tc.want)
			}
		})
	}
}

// trinketWithoutAnEffect returns the lowest-numbered trinket in the
// loaded database that no item effect is already registered for, so the
// test can attach one without colliding with a shipped effect.
func trinketWithoutAnEffect(t *testing.T) int32 {
	t.Helper()
	best := int32(0)
	for id, item := range ItemsByID {
		if item.Type != proto.ItemType_ItemTypeTrinket || HasItemEffect(id) {
			continue
		}
		if best == 0 || id < best {
			best = id
		}
	}
	if best == 0 {
		t.Skip("no effect-free trinket in the item database; this test needs --tags=with_db")
	}
	return best
}

// biomeEnvWearing is setupBiomeEnv with one trinket equipped; an item of
// 0 leaves the slot empty, which is the control.
func biomeEnvWearing(t *testing.T, biome proto.Biome, item int32) *Environment {
	t.Helper()
	target := googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
	equipment := &proto.EquipmentSpec{}
	if item != 0 {
		items := make([]*proto.ItemSpec, int(proto.ItemSlot_ItemSlotRanged)+1)
		for i := range items {
			items[i] = &proto.ItemSpec{}
		}
		items[proto.ItemSlot_ItemSlotTrinket1] = &proto.ItemSpec{Id: item}
		equipment.Items = items
	}
	env, _, _ := NewEnvironment(
		SinglePlayerRaidProto(&proto.Player{
			Name:      "Biome Test",
			Race:      proto.Race_RaceOrc,
			Class:     proto.Class_ClassShaman,
			Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment: equipment,
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		&proto.Encounter{Duration: 180, Biome: biome, Targets: []*proto.Target{target}},
		false,
	)
	return env
}
