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

	env := setupMobTypeDamageEnv(t, itemID, nil, proto.MobType_MobTypeBeast, proto.MobType_MobTypeHumanoid)

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

// A creature-type item effect is set up once, before the pull, and
// permanently mutates attack tables keyed by UnitIndex. It therefore
// means "every target in this fight", not "the targets that are up right
// now", so it has to read the target pool. Reading the active prefix
// instead silently skipped every add on a timeline that starts below the
// pool size - and made the item disagree with racials.go's Troll Beast
// Slaying, which its own doc comment says it mirrors and which has
// always read the pool.
func TestMobTypeDamageEffectCoversTargetsTheTimelineHasNotActivatedYet(t *testing.T) {
	itemID := firstUnregisteredItemIDForTest(t)
	const multiplier = 1.5
	NewMobTypeDamageEffect(itemID, []proto.MobType{proto.MobType_MobTypeBeast}, multiplier)

	// The Beast is second, so the timeline's opening count of one leaves
	// it pooled but inactive while the item effect is being set up.
	env := setupMobTypeDamageEnv(t, itemID,
		[]*proto.TargetCountAt{{AtSeconds: 0, Count: 1}},
		proto.MobType_MobTypeHumanoid, proto.MobType_MobTypeBeast)

	if got := len(env.Encounter.TargetUnits); got != 1 {
		t.Fatalf("active prefix during setup = %d, want the timeline's opening count of 1", got)
	}

	character := env.Raid.Parties[0].Players[0].GetCharacter()
	beast := env.Encounter.AllTargetUnits[1]
	table := character.AttackTables[beast.UnitIndex][proto.CastType_CastTypeMainHand]
	if table.DamageDealtMultiplier != multiplier {
		t.Errorf("DamageDealtMultiplier against the pooled Beast add = %v, want %v", table.DamageDealtMultiplier, multiplier)
	}
}

// setupMobTypeDamageEnv builds a single-player environment with the given
// item equipped in the trinket slot, against one target per mob type, so a
// creature-type-conditional effect can be checked against both a match and
// a non-match in one fight. A non-nil timeline makes only its opening
// count active while the effect is set up.
func setupMobTypeDamageEnv(t *testing.T, itemID int32, timeline []*proto.TargetCountAt, mobTypes ...proto.MobType) *Environment {
	t.Helper()

	targets := make([]*proto.Target, len(mobTypes))
	for i, mobType := range mobTypes {
		targets[i] = googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
		targets[i].MobType = mobType
	}

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
		&proto.Encounter{Duration: 180, Targets: targets, TargetsOverTime: timeline},
		false,
	)
	return env
}
