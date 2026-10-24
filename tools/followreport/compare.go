package main

import "sort"

// Verdict words, written as the report's first line and used as exit codes' meaning.
const (
	verdictClean  = "clean"
	verdictReview = "review"
)

// Keys of ClassDelta.Counts.
const (
	countMismatch     = "mismatch"
	countDiffers      = "declared, differs"
	countUnclassified = "unclassified"
)

// RowChange is a spell row that got worse between two builds.
type RowChange struct {
	Key           RowKey
	Before, After Row
	Added         bool
}

// ClassDelta is everything that moved in one class's golden.
type ClassDelta struct {
	Worsened      []RowChange
	Counts        map[string][2]int
	NewUndisposed []string
}

func isWorse(before, after Row, existed bool) bool {
	if after.Verdict == verdictMismatch && (!existed || before.Verdict != verdictMismatch) {
		return true
	}
	return after.Damage == damageDeclaredDiffer && (!existed || before.Damage != damageDeclaredDiffer)
}

func worsenedRows(before, after Golden) []RowChange {
	var out []RowChange
	for key, a := range after.Rows {
		b, existed := before.Rows[key]
		if isWorse(b, a, existed) {
			out = append(out, RowChange{Key: key, Before: b, After: a, Added: !existed})
		}
	}
	sort.Slice(out, func(i, j int) bool { return rowKeyLess(out[i].Key, out[j].Key) })
	return out
}

func rowKeyLess(a, b RowKey) bool {
	if a.Section != b.Section {
		return a.Section < b.Section
	}
	if a.Spec != b.Spec {
		return a.Spec < b.Spec
	}
	if a.Level != b.Level {
		return a.Level < b.Level
	}
	if a.SpellID != b.SpellID {
		return a.SpellID < b.SpellID
	}
	return a.Rank < b.Rank
}

func countRows(g Golden) map[string]int {
	counts := map[string]int{}
	for _, r := range g.Rows {
		if r.Verdict == verdictMismatch {
			counts[countMismatch]++
		}
		if r.Damage == damageDeclaredDiffer {
			counts[countDiffers]++
		}
	}
	for _, t := range g.Trainables {
		if isUndisposed(t) {
			counts[countUnclassified]++
		}
	}
	return counts
}

// isUndisposed is true for a trainable nobody has classified: either the
// class has no Disposition column at all, or the row says "unclassified".
func isUndisposed(t Trainable) bool {
	return t.Disposition == dispositionNone || t.Disposition == dispositionUnsorted
}

func newUndisposed(before, after Golden) []string {
	var out []string
	for name, t := range after.Trainables {
		if _, existed := before.Trainables[name]; !existed && isUndisposed(t) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// CompareClass diffs one class's golden before and after a follow.
func CompareClass(before, after Golden) ClassDelta {
	b, a := countRows(before), countRows(after)
	counts := map[string][2]int{}
	for _, name := range []string{countMismatch, countDiffers, countUnclassified} {
		counts[name] = [2]int{b[name], a[name]}
	}
	return ClassDelta{
		Worsened:      worsenedRows(before, after),
		Counts:        counts,
		NewUndisposed: newUndisposed(before, after),
	}
}

func (d ClassDelta) needsReview() bool {
	return len(d.Worsened) > 0 || len(d.NewUndisposed) > 0
}

// ClassSetChange lists the classes that appeared in or vanished from the goldens.
type ClassSetChange struct{ Added, Removed []string }

// Empty reports whether the same classes are present on both sides.
func (c ClassSetChange) Empty() bool { return len(c.Added) == 0 && len(c.Removed) == 0 }

// DiffClassSets compares the class names found before and after.
func DiffClassSets(before, after []string) ClassSetChange {
	return ClassSetChange{Added: missingFrom(before, after), Removed: missingFrom(after, before)}
}

func missingFrom(have, want []string) []string {
	set := map[string]bool{}
	for _, h := range have {
		set[h] = true
	}
	var out []string
	for _, w := range want {
		if !set[w] {
			out = append(out, w)
		}
	}
	sort.Strings(out)
	return out
}

// Verdict names whether a follow needs a person: "clean" or "review".
func Verdict(classes map[string]ClassDelta, t TalentDelta, r []RacialChange, buildLog string, classesBefore, classesAfter []string) string {
	review := buildLog != "" || !t.Empty() || len(r) > 0 || !DiffClassSets(classesBefore, classesAfter).Empty()
	for _, d := range classes {
		review = review || d.needsReview()
	}
	if review {
		return verdictReview
	}
	return verdictClean
}

// moved reports whether anything in the class differs, counts included.
func (d ClassDelta) moved() bool {
	if d.needsReview() {
		return true
	}
	for _, c := range d.Counts {
		if c[0] != c[1] {
			return true
		}
	}
	return false
}
