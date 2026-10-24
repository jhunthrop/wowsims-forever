package main

import (
	"strings"
	"testing"
)

func TestRenderFirstLine(t *testing.T) {
	out := Render(Report{Verdict: verdictClean})
	if !strings.HasPrefix(out, "verdict: clean\n") {
		t.Fatalf("output = %q", out)
	}
	if strings.Contains(out, headingBuild) {
		t.Error("clean report must not have a Build section")
	}
}

func TestRenderBuildLogComesBeforeClasses(t *testing.T) {
	out := Render(Report{
		Verdict:  verdictReview,
		BuildLog: "undefined: foo",
		Classes:  map[string]ClassDelta{"mage": {Worsened: []RowChange{{Key: RowKey{Section: sectionSpells, SpellID: 1}}}}},
	})
	if !strings.HasPrefix(out, "verdict: review\n") {
		t.Fatalf("output = %q", out)
	}
	build, class := strings.Index(out, headingBuild), strings.Index(out, "## Class mage")
	if build < 0 || class < 0 || build > class {
		t.Fatalf("Build at %d, class at %d\n%s", build, class, out)
	}
	if !strings.Contains(out, "undefined: foo") {
		t.Error("build log text missing")
	}
}

func TestRenderSections(t *testing.T) {
	out := Render(Report{
		Verdict:   verdictReview,
		Diffstat:  []string{"3\t1\tsim/mage/x.go", "-\t-\tdata/y.bin"},
		Classes:   map[string]ClassDelta{"mage": {Counts: map[string][2]int{countMismatch: {0, 2}}, NewUndisposed: []string{"Polymorph (118)"}}},
		Talents:   TalentDelta{Added: []TalentChange{{Class: "mage", NodeID: 13, Name: "New Talent"}}},
		Racials:   []RacialChange{{Race: "human", Added: []string{"A2"}, Removed: []string{"A"}}},
		ClassesIn: ClassSetChange{Added: []string{"druid"}, Removed: []string{"rogue"}},
	})
	for _, want := range []string{
		"sim/mage/x.go: +3 -1", "data/y.bin: +- --",
		"mismatch: 0 -> 2", "Polymorph (118)",
		"New Talent", "human", "A2", "druid", "rogue",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}
