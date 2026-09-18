package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Pins the H3 ruling from the Task 4 fix round: research/08-stats.md §7
// leaves the baseline Agility/Intellect-to-Crit conversion "entirely
// unpublished" for the unified model, so every class keeps its pre-merge
// sources — a hybrid that converted both keeps both (stacking, marked
// unconfirmed); a class that converted only one keeps only that one,
// unchanged. All nine classes that wire a base-stat-to-Crit dependency are
// asserted here against the one table (sim/core/base_stats.go), which is
// the single place a later ruling edits instead of the nine
// AddStatDependency call sites across sim/druid, sim/shaman,
// sim/paladin, sim/warlock, sim/hunter, sim/rogue, sim/mage, sim/priest
// and sim/warrior.
func TestCritStatSourcesArePinned(t *testing.T) {
	expected := map[proto.Class]CritStatSources{
		// Hybrids: stack Agility- and Intellect-derived Crit, unconfirmed.
		proto.Class_ClassDruid:   {Agility: true, Intellect: true},
		proto.Class_ClassShaman:  {Agility: true, Intellect: true},
		proto.Class_ClassPaladin: {Agility: true, Intellect: true},
		proto.Class_ClassWarlock: {Agility: true, Intellect: true},
		proto.Class_ClassHunter:  {Agility: true, Intellect: true},
		// Single-source: unchanged from before the merge.
		proto.Class_ClassRogue:   {Agility: true},
		proto.Class_ClassWarrior: {Agility: true},
		proto.Class_ClassMage:    {Intellect: true},
		proto.Class_ClassPriest:  {Intellect: true},
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

// A class absent from ClassCritStatSources (every class that wires a
// base-stat-to-Crit dependency is listed — see TestCritStatSourcesArePinned)
// gets neither dependency, and the helper must not panic on an unlisted
// class.
func TestCritStatSourcesNoOpForUnlistedClass(t *testing.T) {
	character := &Character{}
	AddCritStatDependencies(character, proto.Class_ClassUnknown)

	result := character.SortAndApplyStatDependencies(stats.Stats{
		stats.Agility:   100,
		stats.Intellect: 100,
	})
	if result[stats.Crit] != 0 {
		t.Fatalf("Crit = %v, want 0 for a class with no CritStatSources entry", result[stats.Crit])
	}
}

// A single-source class (e.g. Rogue: Agility only) draws Crit from only
// that stat — the Intellect it also has does not leak in, unlike a
// hybrid's deliberate stacking in TestCritStatSourcesStackForHybrids.
func TestCritStatSourcesSingleSourceDoesNotStack(t *testing.T) {
	class := proto.Class_ClassRogue
	agi, intel := 100.0, 200.0

	character := &Character{}
	AddCritStatDependencies(character, class)

	result := character.SortAndApplyStatDependencies(stats.Stats{
		stats.Agility:   agi,
		stats.Intellect: intel,
	})

	wantCrit := agi * CritPerAgiAtLevel[class] * CritRatingPerCritChance
	if result[stats.Crit] != wantCrit {
		t.Fatalf("Crit = %v, want %v (Agility only; Intellect must not contribute)", result[stats.Crit], wantCrit)
	}
}
