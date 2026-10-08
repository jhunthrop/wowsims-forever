package druid

import "github.com/wowsims/classic/sim/common/clientsetbonus"

// The druid PvP rank sets the client defines for Phase 1, every ItemSet id of
// each name. Their bonuses are flat stats or utility the sim never casts.
var pvpSetIDs = []int32{1722, 1723, 1724, 1735, 1736, 1737, 1748, 1749, 1750, 1762, 1763, 1764}

func init() {
	clientsetbonus.RegisterPvPSets(pvpSetIDs...)
}
