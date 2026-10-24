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

func TestRenderWorsenedRowShowsDiff(t *testing.T) {
	w := RowChange{Key: RowKey{Section: sectionSpells, SpellID: 116}, After: Row{Verdict: verdictMismatch, Diff: "cost: 25 vs 30"}}
	out := Render(Report{Verdict: verdictReview, Classes: map[string]ClassDelta{"mage": {Worsened: []RowChange{w}}}})
	if !strings.Contains(out, "cost: 25 vs 30") {
		t.Fatalf("diff missing:\n%s", out)
	}
}

func TestRenderTalentsShowsText(t *testing.T) {
	d := DiffTalents(loadTalentFixture(t, "talents-prev"), loadTalentFixture(t, "talents-cur"))
	out := Render(Report{Verdict: verdictReview, Talents: d})
	for _, want := range []string{
		`changed mage node 10 Arcane Focus`,
		`rank 1: "Hit +2%" → "Hit +3%"`,
		`max_rank 2 → 3`,
		`removed mage node 12 Old Talent`,
		`rank 1: "Gone"`,
		`added mage node 13 New Talent`,
		`rank 1: "Fresh"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}

func TestDescribeChangeRankTextAndSpellID(t *testing.T) {
	prev := Talent{SpellID: 1, MaxRank: 1, Ranks: []TalentRank{{Description: "old"}}}
	cur := Talent{SpellID: 2, MaxRank: 2, Ranks: []TalentRank{{Description: "new"}, {Description: "second"}}}
	got := strings.Join(describeChange(prev, cur), "\n")
	for _, want := range []string{"max_rank 1 → 2", "spell_id 1 → 2", `rank 1: "old" → "new"`, `rank 2: "" → "second"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if lines := describeChange(prev, prev); lines != nil {
		t.Errorf("unchanged node: lines = %v, want nil", lines)
	}
}
