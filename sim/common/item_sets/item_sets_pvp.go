package item_sets

import "github.com/wowsims/classic/sim/common/clientsetbonus"

// The Blood Guard's and Knight-Lieutenant's sets (ItemSets 1618 to 1636 and
// 1665) the client defines for Phase 1, shared across the classes that wear
// each armour type. Their bonuses are all flat stats. The Highlander and
// Defiler reputation sets (467 to 473, 483 to 488) have no Phase 1 source
// in the site's loot index any more, so tools/gen_set_rows.py no longer
// emits them and they are not registered.
var pvpSetIDs = []int32{
	1618, 1619, 1620, 1621, 1622, 1623, 1624, 1625, 1626, 1627,
	1628, 1629, 1630, 1631, 1632, 1633, 1634, 1635, 1636, 1665,
}

func init() {
	clientsetbonus.RegisterPvPSets(pvpSetIDs...)
}
