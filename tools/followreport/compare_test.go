package main

import (
	"path/filepath"
	"testing"
)

func mustParse(t *testing.T, fixture, class string) Golden {
	t.Helper()
	g, err := ParseGolden(filepath.Join("testdata", fixture, class+".golden.md"))
	if err != nil {
		t.Fatalf("ParseGolden(%s): %v", fixture, err)
	}
	return g
}

func classOf(fixture string) string {
	if fixture == "rogue-a" || fixture == "rogue-new-trainable" {
		return "rogue"
	}
	return "mage"
}

func TestCompareClassVerdicts(t *testing.T) {
	cases := []struct{ name, before, after, wantVerdict string }{
		{"identical goldens", "mage-a", "mage-a", verdictClean},
		{"a match row becomes mismatch", "mage-a", "mage-one-mismatch", verdictReview},
		{"a new row already match", "mage-a", "mage-plus-match-row", verdictClean},
		{"a row the client dropped", "mage-a", "mage-minus-row", verdictClean},
		{"a new row that is mismatch", "mage-a", "mage-plus-mismatch-row", verdictReview},
		{"declared, matches becomes declared, differs", "mage-a", "mage-damage-differs", verdictReview},
		{"a new trainable in a class with dispositions, unclassified", "mage-a", "mage-new-unclassified", verdictReview},
		{"a new trainable that arrives classified", "mage-a", "mage-new-classified", verdictClean},
		{"a new trainable in a class with no Disposition column", "rogue-a", "rogue-new-trainable", verdictReview},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			class := classOf(tc.before)
			delta := CompareClass(mustParse(t, tc.before, class), mustParse(t, tc.after, class))
			got := Verdict(map[string]ClassDelta{class: delta}, TalentDelta{}, nil, "", []string{class}, []string{class})
			if got != tc.wantVerdict {
				t.Fatalf("verdict = %s, want %s (delta %+v)", got, tc.wantVerdict, delta)
			}
		})
	}
}

func TestCompareClassCounts(t *testing.T) {
	delta := CompareClass(mustParse(t, "mage-a", "mage"), mustParse(t, "mage-one-mismatch", "mage"))
	if got := delta.Counts[countMismatch]; got != [2]int{0, 1} {
		t.Fatalf("mismatch count = %v, want [0 1]", got)
	}
	if len(delta.Worsened) != 1 || delta.Worsened[0].Key.SpellID != 116 {
		t.Fatalf("worsened = %+v, want exactly Frostbolt", delta.Worsened)
	}
	delta = CompareClass(mustParse(t, "mage-a", "mage"), mustParse(t, "mage-new-unclassified", "mage"))
	if got := delta.Counts[countUnclassified]; got != [2]int{0, 1} {
		t.Fatalf("unclassified count = %v, want [0 1]", got)
	}
	if len(delta.NewUndisposed) != 1 || delta.NewUndisposed[0] != "Polymorph (118)" {
		t.Fatalf("NewUndisposed = %v", delta.NewUndisposed)
	}
}

func TestVerdictClassAddedOrRemoved(t *testing.T) {
	both := []string{"mage", "rogue"}
	one := []string{"mage"}
	if got := Verdict(nil, TalentDelta{}, nil, "", both, one); got != verdictReview {
		t.Fatalf("class removed: %s", got)
	}
	if got := Verdict(nil, TalentDelta{}, nil, "", one, both); got != verdictReview {
		t.Fatalf("class added: %s", got)
	}
	if got := Verdict(nil, TalentDelta{}, nil, "", both, both); got != verdictClean {
		t.Fatalf("same classes: %s", got)
	}
}

func TestVerdictBuildLogIsReview(t *testing.T) {
	if got := Verdict(nil, TalentDelta{}, nil, "compile error", nil, nil); got != verdictReview {
		t.Fatalf("verdict = %s, want review", got)
	}
}

func TestVerdictTalentsAndRacialsAreReview(t *testing.T) {
	td := TalentDelta{Added: []TalentChange{{Class: "mage", NodeID: 1}}}
	if got := Verdict(nil, td, nil, "", nil, nil); got != verdictReview {
		t.Fatalf("talent change: %s", got)
	}
	rc := []RacialChange{{Race: "human", Added: []string{"x"}}}
	if got := Verdict(nil, TalentDelta{}, rc, "", nil, nil); got != verdictReview {
		t.Fatalf("racial change: %s", got)
	}
}
