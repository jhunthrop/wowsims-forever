package clientsetbonustest

import (
	"slices"
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// AssertPvPSetsMatchClient fails unless every id is a registered item set
// whose name and bonus thresholds are the client row's.
func AssertPvPSetsMatchClient(t *testing.T, ids []int32) {
	t.Helper()
	registered := map[int32]*core.ItemSet{}
	for _, set := range core.RegisteredItemSets() {
		registered[set.ID] = set
	}
	for _, id := range ids {
		row, ok := core.ClientSetRow(id)
		if !ok {
			t.Errorf("set %d has no client row", id)
			continue
		}
		set, ok := registered[id]
		if !ok {
			t.Errorf("%s (%d) is not registered", row.Name, id)
			continue
		}
		if set.Name != row.Name {
			t.Errorf("set %d is registered as %q, the client calls it %q", id, set.Name, row.Name)
		}
		if got, want := registeredThresholds(set), clientThresholds(row); !slices.Equal(got, want) {
			t.Errorf("%s (%d) has bonuses at %v pieces, the client has %v", row.Name, id, got, want)
		}
	}
}

func registeredThresholds(set *core.ItemSet) []int32 {
	thresholds := make([]int32, 0, len(set.Bonuses))
	for threshold := range set.Bonuses {
		thresholds = append(thresholds, threshold)
	}
	slices.Sort(thresholds)
	return thresholds
}

func clientThresholds(row core.ClientSet) []int32 {
	thresholds := make([]int32, 0, len(row.Bonuses))
	for _, bonus := range row.Bonuses {
		thresholds = append(thresholds, bonus.Threshold)
	}
	slices.Sort(thresholds)
	return thresholds
}
