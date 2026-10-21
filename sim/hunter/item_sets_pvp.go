package hunter

import "github.com/wowsims/classic/sim/common/clientsetbonus"

// The hunter PvP rank sets the client defines for Phase 1, every ItemSet id of
// each name. Their bonuses are flat stats or utility the sim never casts.
var pvpSetIDs = []int32{361, 543, 550, 1725, 1726, 1738, 1739, 1751, 1752, 1765, 1766}

func init() {
	clientsetbonus.RegisterPvPSets(pvpSetIDs...)
}
