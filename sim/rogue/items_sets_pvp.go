package rogue

import "github.com/wowsims/classic/sim/common/clientsetbonus"

// The rogue PvP rank sets the client defines for Phase 1, every ItemSet id of
// each name. Their bonuses are flat stats or utility the sim never casts.
var pvpSetIDs = []int32{347, 548, 1730, 1743, 1756, 1770}

func init() {
	clientsetbonus.RegisterPvPSets(pvpSetIDs...)
}
