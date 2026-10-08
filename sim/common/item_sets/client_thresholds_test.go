package item_sets_test

import (
	"slices"
	"testing"

	_ "github.com/wowsims/classic/sim" // imports every class package, so every client set is registered.
	"github.com/wowsims/classic/sim/core"
)

func registeredSetsByID() map[int32]*core.ItemSet {
	byID := map[int32]*core.ItemSet{}
	for _, set := range core.RegisteredItemSets() {
		byID[set.ID] = set
	}
	return byID
}

func clientThresholds(row core.ClientSet) []int32 {
	var thresholds []int32
	for _, bonus := range row.Bonuses {
		if !slices.Contains(thresholds, bonus.Threshold) {
			thresholds = append(thresholds, bonus.Threshold)
		}
	}
	slices.Sort(thresholds)
	return thresholds
}

func registeredThresholds(set *core.ItemSet) []int32 {
	thresholds := make([]int32, 0, len(set.Bonuses))
	for threshold := range set.Bonuses {
		thresholds = append(thresholds, threshold)
	}
	slices.Sort(thresholds)
	return thresholds
}

// Forever re-itemised its sets, so a set registered from the client's rows
// has exactly the client's thresholds and no vanilla leftovers.
func TestClientSetsHaveTheClientThresholds(t *testing.T) {
	registered := registeredSetsByID()
	if len(core.ClientSetModels()) == 0 {
		t.Fatal("no set was registered through NewClientItemSet")
	}
	for id := range core.ClientSetModels() {
		row, ok := core.ClientSetRow(id)
		if !ok {
			t.Errorf("set %d is registered but has no client row", id)
			continue
		}
		set, ok := registered[id]
		if !ok {
			t.Errorf("set %d (%s) has client models but is not registered", id, row.Name)
			continue
		}
		want, got := clientThresholds(row), registeredThresholds(set)
		if !slices.Equal(want, got) {
			t.Errorf("set %d (%s): thresholds %v, the client has %v", id, row.Name, got, want)
		}
	}
}

// A set registers once: a second registration under the same id would
// leave the vanilla thresholds reachable by name.
func TestClientSetsRegisterOnce(t *testing.T) {
	seen := map[int32]bool{}
	for _, set := range core.RegisteredItemSets() {
		if set.ID == 0 {
			continue
		}
		if seen[set.ID] {
			t.Errorf("set %d (%s) is registered twice", set.ID, set.Name)
		}
		seen[set.ID] = true
	}
}
