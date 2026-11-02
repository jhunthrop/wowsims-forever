package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

func buildHunterWearing(t *testing.T, slot proto.ItemSlot, itemID int32) *Hunter {
	t.Helper()
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotRanged] = &proto.ItemSpec{Id: scopeTestGun}
	if itemID != 0 {
		items[slot] = &proto.ItemSpec{Id: itemID}
	}
	player := core.WithSpec(
		&proto.Player{
			Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, Level: 60,
			Equipment: &proto.EquipmentSpec{Items: items},
			Buffs:     &proto.IndividualBuffs{},
		},
		&proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{PetType: proto.Hunter_Options_PetNone}}},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
}

// Four classic-db items link, in the 1.60.1.70707 client, to "Increased Hit
// Chance - Ranged" (aura 54) restricted to ranged weapons (SpellEquippedItems
// subclass mask 262156): Trueaim Gauntlets 13255 to 1301127 (3%), and
// Infernal Trickster Leggings 17754, Marksman Bands 18296 and Stoneshatter
// 18388 to 1294769 (1%). The site's simdb carries none of it; ranged attacks
// gain it and melee attacks do not. (The fork's own database still carries
// Trueaim's Era 1% hit; the site's simdb, which the published sims read, does
// not.)
func TestForeverRangedHitItemsReachRangedAttacksOnly(t *testing.T) {
	base := buildHunterWearing(t, proto.ItemSlot_ItemSlotHands, 0)
	for _, tc := range []struct {
		item int32
		slot proto.ItemSlot
		hit  float64
	}{
		{13255, proto.ItemSlot_ItemSlotHands, 3},
		{17754, proto.ItemSlot_ItemSlotLegs, 1},
		{18296, proto.ItemSlot_ItemSlotWrist, 1},
		{18388, proto.ItemSlot_ItemSlotRanged, 1},
	} {
		h := buildHunterWearing(t, tc.slot, tc.item)
		want := tc.hit * core.HitRatingPerHitChance
		if got := h.AutoAttacks.RangedAuto().BonusHitRating - base.AutoAttacks.RangedAuto().BonusHitRating; got != want {
			t.Errorf("item %d: Auto Shot hit bonus %v, want %v", tc.item, got, want)
		}
		if got := h.AutoAttacks.MHAuto().BonusHitRating - base.AutoAttacks.MHAuto().BonusHitRating; got != 0 {
			t.Errorf("item %d: melee swing hit bonus %v, want 0", tc.item, got)
		}
	}
}
