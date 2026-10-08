package warrior

import "github.com/wowsims/classic/sim/common/clientsetbonus"

// The warrior PvP rank sets the client defines for Phase 1, every ItemSet id of
// each name. Their bonuses are flat stats or utility the sim never casts.
var pvpSetIDs = []int32{537, 545, 1721, 1747, 1761, 1775}

func init() {
	clientsetbonus.RegisterPvPSets(pvpSetIDs...)
}
