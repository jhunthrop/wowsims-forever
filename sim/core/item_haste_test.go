package core

import (
	"math"
	"testing"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// An item's haste counts once. simdb states an item's "attack speed and
// casting speed by N%" (Shard of the Defiler 17142: 15 and 15) as MeleeHaste
// and SpellHaste in percent (HasteRatingPerHastePercent is 1). The equipment's
// stats reach the unit's stats, which the swing and cast speed read; the
// engine also multiplied the speed multipliers by the same item stats, so a
// 15% item swung and cast 32.25% faster.
func TestAnItemsHasteCountsOnce(t *testing.T) {
	const hasteItem = 990017142
	addToDatabase(&proto.SimDatabase{Items: []*proto.SimItem{{
		Id:              hasteItem,
		Type:            proto.ItemType_ItemTypeWeapon,
		HandType:        proto.HandType_HandTypeMainHand,
		WeaponType:      proto.WeaponType_WeaponTypeDagger,
		WeaponDamageMin: 52,
		WeaponDamageMax: 61,
		WeaponSpeed:     1.1,
		Stats:           stats.Stats{stats.MeleeHaste: 15, stats.SpellHaste: 15}.ToFloatArray(),
	}}})
	target := googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
	env, _, _ := NewEnvironment(
		SinglePlayerRaidProto(&proto.Player{
			Name:      "Haste Test",
			Race:      proto.Race_RaceHuman,
			Class:     proto.Class_ClassShaman,
			Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{{Id: hasteItem}}},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		&proto.Encounter{Duration: 180, Targets: []*proto.Target{target}},
		false,
	)
	sim := newSimWithEnv(env, &proto.SimOptions{RandomSeed: 1}, simsignals.CreateSignals())
	sim.reset()
	character := env.Raid.Parties[0].Players[0].GetCharacter()
	if got := character.SwingSpeed(); math.Abs(got-1.15) > 1e-9 {
		t.Errorf("swing speed %.4f, want 1.15", got)
	}
	if got := 1 / character.CastSpeed; math.Abs(got-1.15) > 1e-9 {
		t.Errorf("cast speed %.4f, want 1.15", got)
	}
}
