package item_sets

import "github.com/wowsims/classic/sim/common/clientsetbonus"

// The Blood Guard's and Knight-Lieutenant's sets (ItemSets 1618 to 1636 and
// 1665) and the Highlander and Defiler reputation sets (467 to 473, 483 to
// 488) the client defines for Phase 1, shared across the classes that wear
// each armour type. Their bonuses are all flat stats.
var pvpSetIDs = []int32{
	1618, 1619, 1620, 1621, 1622, 1623, 1624, 1625, 1626, 1627,
	1628, 1629, 1630, 1631, 1632, 1633, 1634, 1635, 1636, 1665,
	467, 468, 469, 470, 471, 472, 473, 483, 484, 485, 486, 487, 488,
}

func init() {
	clientsetbonus.RegisterPvPSets(pvpSetIDs...)
}
