package warlock

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// The warlock PvP rank sets (client ItemSets), every id of each name:
// Champion's and Lieutenant Commander's Dreadgear, then the Threads sets.
// Each has the Immolate bonus; the rest is flat.
const (
	immolateCastTimeBonus = 23047
	immolateClientFamily  = 0x4 // word 0 of the bonus spell's family mask
)

var immolateClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{immolateClientFamily}, Engine: WarlockSpellMaskImmolate},
}

// applyImmolateCastTime is the bonus's cast time and global cooldown
// reductions. The row carries both modifiers although its text names only
// the cast time.
func applyImmolateCastTime(agent core.Agent) {
	clientsetbonus.AddStaticMods(&agent.GetCharacter().Unit, immolateCastTimeBonus, immolateClassMasks)
}

var pvpSetIDs = []int32{541, 547, 1760, 1774, 1734, 1746}

func init() {
	for _, id := range pvpSetIDs {
		clientsetbonus.RegisterPvPSet(id, map[int32]core.ApplyEffect{immolateCastTimeBonus: applyImmolateCastTime})
	}
}
