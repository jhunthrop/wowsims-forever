package hunter

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// The ability masks a Tier 1 bonus modifies.
const (
	HunterSpellMaskAimedShot uint64 = 1 << iota
	HunterSpellMaskMultiShot
)

// tier1ClassMasks says which engine spells the family of the Hunter 5P row
// is. The row names the Aimed Shot family (client bit 131072) only, but its
// text, "Aimed Shot and Multi-Shot", and the owner's brief name both, and
// Multi-Shot's own family (4096) is in no Tier 1 row: the bonus is modelled
// on both abilities.
var tier1ClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{131072}, Engine: HunterSpellMaskAimedShot | HunterSpellMaskMultiShot},
}

// Wildstalker Armor is Forever's Tier 1 hunter set (client ItemSet 2101,
// build 1.60.1.70009). The 2-piece haste and the 4-piece attack power
// against Beasts are flat bonuses applied from the client's rows.
const (
	wildstalkerArmorSetID int32 = 2101
	wildstalkerTrapsBonus int32 = 1301012
	wildstalkerShotsBonus int32 = 1301252
)

var ItemSetWildstalkerArmor = core.NewClientItemSet(core.ClientSetModel{
	ID: wildstalkerArmorSetID,
	Effects: map[int32]core.ApplyEffect{
		// Reduces the cooldown on your Aimed Shot and Multi-Shot abilities by 1 sec.
		wildstalkerShotsBonus: clientsetbonus.StaticMod(wildstalkerArmorSetID, clientsetbonus.FivePieces, tier1ClassMasks),
	},
	NoSim: map[int32]string{
		wildstalkerTrapsBonus: "reduces the cooldown of Frost Trap and Freezing Trap, crowd control the sim never casts",
	},
})
