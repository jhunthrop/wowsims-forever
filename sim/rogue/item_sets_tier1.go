package rogue

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// Grimstitch Armor is Forever's Tier 1 rogue set (client ItemSet 2099,
// build 1.60.1.70009). The 2-piece hit and the 4-piece attack power against
// Humanoids are flat bonuses applied from the client's rows.
const (
	grimstitchArmorSetID        int32 = 2099
	grimstitchKidneyShotBonus   int32 = 1301047
	grimstitchFinisherCostBonus int32 = 1301708
)

var ItemSetGrimstitchArmor = core.NewClientItemSet(core.ClientSetModel{
	ID: grimstitchArmorSetID,
	Effects: map[int32]core.ApplyEffect{
		// Reduces the cost of your Envenom and Eviscerate abilities by 5 Energy.
		grimstitchFinisherCostBonus: clientsetbonus.StaticMod(grimstitchArmorSetID, clientsetbonus.FivePieces, rogueClassMasks),
	},
	NoSim: map[int32]string{
		grimstitchKidneyShotBonus: "reduces the cooldown of Kidney Shot, a utility stun the sim never casts",
	},
})
