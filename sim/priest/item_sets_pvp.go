package priest

import "github.com/wowsims/classic/sim/common/clientsetbonus"

// The priest PvP rank sets the client defines for Phase 1, every ItemSet id of
// each name. Their bonuses are flat stats or utility the sim never casts.
var pvpSetIDs = []int32{342, 540, 1728, 1729, 1741, 1742, 1754, 1755, 1768, 1769}

func init() {
	clientsetbonus.RegisterPvPSets(pvpSetIDs...)
}
