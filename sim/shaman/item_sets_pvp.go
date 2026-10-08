package shaman

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// shockCritBonus is the PvP rank sets' "Improves your chance to get a
// critical strike with all Shock spells" bonus spell.
const shockCritBonus = 22804

// The client families of the shocks: Earth Shock, Flame Shock (the same
// bit relicClassMasks uses) and Frost Shock.
var shockClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{1 << 20}, Engine: ShamanSpellMaskEarthShock},
	{Client: core.ClientClassMask{1 << 28}, Engine: ShamanSpellMaskFlameShock},
	{Client: core.ClientClassMask{1 << 31}, Engine: ShamanSpellMaskFrostShock},
}

func applyShockCrit(agent core.Agent) {
	clientsetbonus.AddStaticMods(&agent.GetCharacter().Unit, shockCritBonus, shockClassMasks)
}

// The shaman PvP rank sets (client ItemSets), every one with the shock crit
// bonus: Champion's Stormcaller 538, Thunderfist 1757, Wartide 1758 and
// Earthshaker 1759; Warlord's Earthshaker 1731, Thunderfist 1732 and Wartide
// 1733.
var pvpSetIDs = []int32{538, 1757, 1758, 1759, 1731, 1732, 1733}

func init() {
	for _, id := range pvpSetIDs {
		clientsetbonus.RegisterPvPSet(id, map[int32]core.ApplyEffect{shockCritBonus: applyShockCrit})
	}
}
