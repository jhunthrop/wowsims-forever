package warrior

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// tier1ClassMasks says which engine spell each family of the Warrior 5P
// rows is: Recklessness (client bit 16) and Shield Wall (client bit 8192).
var tier1ClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{16}, Engine: WarriorSpellMaskRecklessness},
	{Client: core.ClientClassMask{8192}, Engine: WarriorSpellMaskShieldWall},
}

// Battlegear of Glory is Forever's Tier 1 Arms and Fury warrior set (client
// ItemSet 2102, build 1.60.1.70009). The 2-piece haste and the 4-piece
// attack power against Humanoids are flat bonuses applied from the
// client's rows.
const (
	battlegearOfGlorySetID      int32 = 2102
	battlegearIntimidatingShout int32 = 1301069
	battlegearRecklessnessBonus int32 = 1301721
	battleplateOfGlorySetID     int32 = 2103
	battleplateIntervene        int32 = 1301071
	battleplateShieldWallBonus  int32 = 1301722
)

var ItemSetBattlegearOfGlory = core.NewClientItemSet(core.ClientSetModel{
	ID: battlegearOfGlorySetID,
	Effects: map[int32]core.ApplyEffect{
		// Reduces the cooldown on your Recklessness ability by 30 sec.
		battlegearRecklessnessBonus: clientsetbonus.StaticMod(battlegearOfGlorySetID, clientsetbonus.FivePieces, tier1ClassMasks),
	},
	NoSim: map[int32]string{
		battlegearIntimidatingShout: "reduces the cooldown of Intimidating Shout, crowd control the sim never casts",
	},
})

// Battleplate of Glory is Forever's Tier 1 Protection warrior set (client
// ItemSet 2103). The 2-piece defense and the 4-piece expertise are flat
// bonuses applied from the client's rows.
var ItemSetBattleplateOfGlory = core.NewClientItemSet(core.ClientSetModel{
	ID: battleplateOfGlorySetID,
	Effects: map[int32]core.ApplyEffect{
		// Reduces the cooldown on your Shield Wall ability by 30 sec.
		battleplateShieldWallBonus: clientsetbonus.StaticMod(battleplateOfGlorySetID, clientsetbonus.FivePieces, tier1ClassMasks),
	},
	NoSim: map[int32]string{
		battleplateIntervene: "reduces the cooldown of Intervene, a movement ability the sim never casts",
	},
})
