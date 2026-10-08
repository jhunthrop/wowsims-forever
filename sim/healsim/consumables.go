package healsim

import "github.com/wowsims/classic/sim/core/proto"

// The item ids of the two mana consumables a raid healer carries.
const (
	MajorManaPotionItem int32 = 13444
	DemonicRuneItem     int32 = 12662
)

// ManaConsumables is the Major Mana Potion and the Demonic Rune, the pair
// the site's raid preset gives every healer.
func ManaConsumables() *proto.Consumes {
	return &proto.Consumes{
		DefaultPotion:   proto.Potions_MajorManaPotion,
		DefaultConjured: proto.Conjured_ConjuredDemonicRune,
	}
}

// ManaGainedFromItem is the mana the healer took in from the item over the
// run, summed over its iterations.
func ManaGainedFromItem(player *proto.UnitMetrics, itemID int32) float64 {
	var gained float64
	for _, resource := range player.Resources {
		if resource.Type == proto.ResourceType_ResourceTypeMana && resource.Id.GetItemId() == itemID {
			gained += resource.ActualGain
		}
	}
	return gained
}

// UnusedManaConsumables names the mana consumables of ManaConsumables the
// healer never used, so a test can say which one its rotation left in the
// bag.
func UnusedManaConsumables(player *proto.UnitMetrics) []string {
	var unused []string
	if ManaGainedFromItem(player, MajorManaPotionItem) <= 0 {
		unused = append(unused, "Major Mana Potion")
	}
	if ManaGainedFromItem(player, DemonicRuneItem) <= 0 {
		unused = append(unused, "Demonic Rune")
	}
	return unused
}
