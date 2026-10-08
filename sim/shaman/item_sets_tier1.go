package shaman

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// Forever's Tier 1 shaman sets (client ItemSets 2109, 2110 and 2111, build
// 1.60.1.70009). The 2- and 4-piece bonuses are flat and applied from the
// client's rows; the 5-piece bonuses are spell cooldown mods.
const (
	spiritcallerSetID       int32 = 2109
	spiritcallersRageSetID  int32 = 2110
	spiritcallersStormSetID int32 = 2111

	fireAndWaterTotemRadiusNoSimNote = "widens the radius of beneficial Fire and Water totems; totem range is not simulated"
	airAndEarthTotemRadiusNoSimNote  = "widens the radius of beneficial Air and Earth totems; totem range is not simulated"
	earthbindTotemNoSimNote          = "reduces the cooldown of Earthbind Totem, a utility totem the sim never drops"
)

// tier1ClassMasks says which engine spells the client spell families of
// the Tier 1 5-piece bonuses are (the SpellClassOptions masks of Riptide,
// Stormstrike and Lava Burst).
var tier1ClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{0, 0, 1 << 4}, Engine: ShamanSpellMaskRiptide},
	{Client: core.ClientClassMask{0, 1 << 4}, Engine: ShamanSpellMaskStormstrike},
	{Client: core.ClientClassMask{0, 1 << 12}, Engine: ShamanSpellMaskLavaBurst},
}

var ItemSetTheSpiritcaller = core.NewClientItemSet(core.ClientSetModel{
	ID: spiritcallerSetID,
	// Reduces the cooldown on your Riptide spell by 1 sec.
	Effects: clientsetbonus.FivePieceMod(spiritcallerSetID, tier1ClassMasks),
	NoSim:   clientsetbonus.NoSimThreePiece(spiritcallerSetID, fireAndWaterTotemRadiusNoSimNote),
})

var ItemSetTheSpiritcallersRage = core.NewClientItemSet(core.ClientSetModel{
	ID: spiritcallersRageSetID,
	// Reduces the cooldown on your Stormstrike ability by 0.5 sec.
	Effects: clientsetbonus.FivePieceMod(spiritcallersRageSetID, tier1ClassMasks),
	NoSim:   clientsetbonus.NoSimThreePiece(spiritcallersRageSetID, airAndEarthTotemRadiusNoSimNote),
})

var ItemSetTheSpiritcallersStorm = core.NewClientItemSet(core.ClientSetModel{
	ID: spiritcallersStormSetID,
	// Reduces the cooldown on your Lava Burst spell by 1 sec.
	Effects: clientsetbonus.FivePieceMod(spiritcallersStormSetID, tier1ClassMasks),
	NoSim:   clientsetbonus.NoSimThreePiece(spiritcallersStormSetID, earthbindTotemNoSimNote),
})
