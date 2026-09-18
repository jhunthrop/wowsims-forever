package core

import (
	"testing"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/classic/sim/core/proto"
)

// NewMobTypeDamageEffect mutates character.AttackTables[...], which is only
// allocated later, by Environment.setupAttackTables() during finalize().
// Item effects run earlier, during initialize() (Character.applyItemEffects,
// called from Raid.applyCharacterEffects). A constructor that mutates
// AttackTables synchronously inside the effect closure panics with "index
// out of range" the moment any item equips it, before a single sim
// iteration runs. This test builds a real environment, equips a real item
// that uses NewMobTypeDamageEffect, runs it through to finalize, and checks
// the multiplier landed on the matching-creature-type target and not on the
// non-matching one.
func TestMobTypeDamageEffectAppliesOnlyToMatchingCreatureType(t *testing.T) {
	itemID := firstUnregisteredItemIDForTest(t)
	const multiplier = 1.5
	NewMobTypeDamageEffect(itemID, []proto.MobType{proto.MobType_MobTypeBeast}, multiplier)

	env := setupMobTypeDamageEnv(t, itemID)

	character := env.Raid.Parties[0].Players[0].GetCharacter()
	beastTable := character.AttackTables[env.Encounter.TargetUnits[0].UnitIndex][proto.CastType_CastTypeMainHand]
	humanoidTable := character.AttackTables[env.Encounter.TargetUnits[1].UnitIndex][proto.CastType_CastTypeMainHand]

	if beastTable.DamageDealtMultiplier != multiplier {
		t.Errorf("DamageDealtMultiplier against the Beast target = %v, want %v", beastTable.DamageDealtMultiplier, multiplier)
	}
	if humanoidTable.DamageDealtMultiplier != 1 {
		t.Errorf("DamageDealtMultiplier against the non-matching Humanoid target = %v, want 1 (unaffected)", humanoidTable.DamageDealtMultiplier)
	}
}

// firstUnregisteredItemIDForTest returns a real item ID from the loaded item
// database that has no item effect registered yet, so NewItemEffect (which
// panics on both an unknown ID under --tags=with_db and a double
// registration) accepts it.
func firstUnregisteredItemIDForTest(t *testing.T) int32 {
	t.Helper()
	for id := range ItemsByID {
		if !HasItemEffect(id) {
			return id
		}
	}
	t.Fatal("no unregistered item found in ItemsByID; run with --tags=with_db")
	return 0
}

// setupMobTypeDamageEnv builds a single-player environment with the given
// item equipped in the trinket slot, against two targets of different
// creature types, so a creature-type-conditional effect can be checked
// against both a match and a non-match in one fight.
func setupMobTypeDamageEnv(t *testing.T, itemID int32) *Environment {
	t.Helper()

	beastTarget := googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
	beastTarget.MobType = proto.MobType_MobTypeBeast
	humanoidTarget := googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
	humanoidTarget.MobType = proto.MobType_MobTypeHumanoid

	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotTrinket1] = &proto.ItemSpec{Id: itemID}

	env, _, _ := NewEnvironment(
		SinglePlayerRaidProto(&proto.Player{
			Name:      "Item Effect Test",
			Race:      proto.Race_RaceOrc,
			Class:     proto.Class_ClassShaman,
			Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment: &proto.EquipmentSpec{Items: items},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		&proto.Encounter{Duration: 180, Targets: []*proto.Target{beastTarget, humanoidTarget}},
		false,
	)
	return env
}
