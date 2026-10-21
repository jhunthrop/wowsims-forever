package core

import (
	"fmt"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// clientSetTestSlots is the order synthetic set pieces fill: the eight
// armour slots a set can span, each with the item type that fits it.
var clientSetTestSlots = []struct {
	slot proto.ItemSlot
	kind proto.ItemType
}{
	{proto.ItemSlot_ItemSlotHead, proto.ItemType_ItemTypeHead},
	{proto.ItemSlot_ItemSlotShoulder, proto.ItemType_ItemTypeShoulder},
	{proto.ItemSlot_ItemSlotChest, proto.ItemType_ItemTypeChest},
	{proto.ItemSlot_ItemSlotHands, proto.ItemType_ItemTypeHands},
	{proto.ItemSlot_ItemSlotLegs, proto.ItemType_ItemTypeLegs},
	{proto.ItemSlot_ItemSlotFeet, proto.ItemType_ItemTypeFeet},
	{proto.ItemSlot_ItemSlotWaist, proto.ItemType_ItemTypeWaist},
	{proto.ItemSlot_ItemSlotWrist, proto.ItemType_ItemTypeWrist},
}

// clientSetTestItemBase keeps synthetic ids clear of every real item id.
const clientSetTestItemBase = 9_000_000

// MaxClientSetTestPieces is the most pieces ClientSetTestGear can supply.
var MaxClientSetTestPieces = len(clientSetTestSlots)

// ClientSetTestGear is a test's way to wear a client set: pieces zero-stat
// items that carry the set's id and name, in the first slots of a full
// 17-slot equipment spec, and the database that defines them (pass it as
// proto.Player.Database). The pieces exist in no real database, so a test
// of a set bonus reads exactly that bonus and nothing an item carries.
func ClientSetTestGear(setID int32, pieces int) (*proto.EquipmentSpec, *proto.SimDatabase) {
	row, ok := ClientSetRow(setID)
	if !ok {
		panic(fmt.Sprintf("core: ClientSetTestGear: no client item set %d", setID))
	}
	if pieces < 0 || pieces > len(clientSetTestSlots) {
		panic(fmt.Sprintf("core: ClientSetTestGear: %d pieces; a set spans at most %d slots", pieces, len(clientSetTestSlots)))
	}
	items := make([]*proto.ItemSpec, int(proto.ItemSlot_ItemSlotRanged)+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	db := &proto.SimDatabase{}
	for i := 0; i < pieces; i++ {
		piece := clientSetTestSlots[i]
		id := int32(clientSetTestItemBase) + setID*int32(len(clientSetTestSlots)) + int32(i)
		items[piece.slot] = &proto.ItemSpec{Id: id}
		db.Items = append(db.Items, &proto.SimItem{
			Id:      id,
			Name:    fmt.Sprintf("%s test piece %d", row.Name, i+1),
			Type:    piece.kind,
			SetName: row.Name,
			SetId:   setID,
			Stats:   stats.Stats{}.ToFloatArray(),
		})
	}
	return &proto.EquipmentSpec{Items: items}, db
}
