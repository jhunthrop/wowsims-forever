package paladin

import "github.com/wowsims/classic/sim/common/clientsetbonus"

// The paladin PvP rank sets the client defines for Phase 1, every ItemSet id of
// each name. Their bonuses are flat stats or utility the sim never casts.
var pvpSetIDs = []int32{544, 1744, 1745, 1776, 1777}

func init() {
	clientsetbonus.RegisterPvPSets(pvpSetIDs...)
}
