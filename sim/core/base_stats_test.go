package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Pins the H3 ruling from the Task 4 fix round: research/08-stats.md §7
// leaves the baseline Agility/Intellect-to-Crit conversion "entirely
// unpublished" for the unified model, so every class that converted both
// before the merge keeps both, stacking on the one Crit stat, and this
// table is the single place a later ruling edits instead of the five
// AddStatDependency call sites in sim/druid, sim/shaman, sim/paladin,
// sim/warlock and sim/hunter.
func TestCritStatSourcesArePinned(t *testing.T) {
	expected := map[proto.Class]CritStatSources{
		proto.Class_ClassDruid:   {Agility: true, Intellect: true},
		proto.Class_ClassShaman:  {Agility: true, Intellect: true},
		proto.Class_ClassPaladin: {Agility: true, Intellect: true},
		proto.Class_ClassWarlock: {Agility: true, Intellect: true},
		proto.Class_ClassHunter:  {Agility: true, Intellect: true},
	}

	if len(ClassCritStatSources) != len(expected) {
		t.Fatalf("ClassCritStatSources has %d entries, want %d — a class was added or removed without updating this pin",
			len(ClassCritStatSources), len(expected))
	}
	for class, want := range expected {
		got, ok := ClassCritStatSources[class]
		if !ok {
			t.Errorf("ClassCritStatSources is missing %v", class)
			continue
		}
		if got != want {
			t.Errorf("ClassCritStatSources[%v] = %+v, want %+v", class, got, want)
		}
	}
}

// Demonstrates (and pins) the actual arithmetic: a hybrid class's Agility
// and Intellect both add into the one Crit stat, because
// StatDependencyManager sums every enabled dependency into its
// destination (see TestMultipleStatDep in sim/core/stats/deps_test.go for
// the same behaviour at the mechanism level). If a later ruling on
// research/08-stats.md §7 decides a hybrid should draw from only one
// source, this is the test that must change, alongside
// ClassCritStatSources.
func TestCritStatSourcesStackForHybrids(t *testing.T) {
	class := proto.Class_ClassDruid
	agi, intel := 100.0, 200.0

	character := &Character{}
	AddCritStatDependencies(character, class)

	result := character.SortAndApplyStatDependencies(stats.Stats{
		stats.Agility:   agi,
		stats.Intellect: intel,
	})

	wantCrit := agi*CritPerAgiAtLevel[class]*CritRatingPerCritChance + intel*CritPerIntAtLevel[class]*CritRatingPerCritChance
	if result[stats.Crit] != wantCrit {
		t.Fatalf("Crit = %v, want %v (Agility and Intellect contributions summed)", result[stats.Crit], wantCrit)
	}
}

// A class absent from ClassCritStatSources (e.g. Rogue, Mage, Priest —
// none of which wire a base-stat-to-Crit dependency today) gets neither
// dependency, and the helper must not panic on an unlisted class.
func TestCritStatSourcesNoOpForUnlistedClass(t *testing.T) {
	character := &Character{}
	AddCritStatDependencies(character, proto.Class_ClassRogue)

	result := character.SortAndApplyStatDependencies(stats.Stats{
		stats.Agility:   100,
		stats.Intellect: 100,
	})
	if result[stats.Crit] != 0 {
		t.Fatalf("Crit = %v, want 0 for a class with no CritStatSources entry", result[stats.Crit])
	}
}
