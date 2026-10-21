package item_sets

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
)

func TestPvPSetsHaveTheClientThresholds(t *testing.T) {
	clientsetbonustest.AssertPvPSetsMatchClient(t, pvpSetIDs)
}
