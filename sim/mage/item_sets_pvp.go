package mage

import "github.com/wowsims/classic/sim/common/clientsetbonus"

// The mage PvP rank sets the client defines for Phase 1, every ItemSet id of
// each name. Their bonuses are flat stats or utility the sim never casts.
var pvpSetIDs = []int32{341, 542, 1727, 1740, 1753, 1767}

func init() {
	clientsetbonus.RegisterPvPSets(pvpSetIDs...)
}
