package rogue

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// The ability masks a Tier 1 bonus modifies. The engine has no Envenom, so
// the Envenom half of the 5-piece has nothing to reach.
const (
	RogueSpellMaskEviscerate uint64 = 1 << iota
)

// tier1ClassMasks says which engine spell the family of the Rogue 5P row is
// (Eviscerate's client family bit 131072; the row's second word is
// Envenom's).
var tier1ClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{131072}, Engine: RogueSpellMaskEviscerate},
}

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
		grimstitchFinisherCostBonus: clientsetbonus.StaticMod(grimstitchArmorSetID, clientsetbonus.FivePieces, tier1ClassMasks),
	},
	NoSim: map[int32]string{
		grimstitchKidneyShotBonus: "reduces the cooldown of Kidney Shot, a utility stun the sim never casts",
	},
})
