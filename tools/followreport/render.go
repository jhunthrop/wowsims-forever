package main

import (
	"fmt"
	"sort"
	"strings"
)

// Report headings.
const (
	verdictPrefix      = "verdict: "
	headingBuild       = "## Build"
	headingRegenerated = "## Regenerated files"
	headingClassSet    = "## Classes in the goldens"
	headingTalents     = "## Talent tables"
	headingRacials     = "## Racial text"
	headingClassPrefix = "## Class "
	numstatFields      = 3
)

// Report is everything Render prints.
type Report struct {
	Verdict   string
	BuildLog  string
	Diffstat  []string
	Classes   map[string]ClassDelta
	ClassesIn ClassSetChange
	Talents   TalentDelta
	Racials   []RacialChange
}

type markdown struct{ strings.Builder }

func (m *markdown) section(heading string) { fmt.Fprintf(&m.Builder, "\n%s\n\n", heading) }
func (m *markdown) line(format string, args ...any) {
	fmt.Fprintf(&m.Builder, format+"\n", args...)
}

// Render writes the report as markdown; the first line is the verdict.
func Render(r Report) string {
	var m markdown
	m.line("%s%s", verdictPrefix, r.Verdict)
	if r.BuildLog != "" {
		m.section(headingBuild)
		m.line("The engine did not compile after regeneration:\n\n```\n%s\n```", strings.TrimRight(r.BuildLog, "\n"))
	}
	renderDiffstat(&m, r.Diffstat)
	renderClassSet(&m, r.ClassesIn)
	renderClasses(&m, r.Classes)
	renderTalents(&m, r.Talents)
	renderRacials(&m, r.Racials)
	return m.String()
}

func renderDiffstat(m *markdown, numstat []string) {
	if len(numstat) == 0 {
		return
	}
	m.section(headingRegenerated)
	for _, line := range numstat {
		fields := strings.SplitN(line, "\t", numstatFields)
		if len(fields) != numstatFields {
			m.line("- %s", line)
			continue
		}
		m.line("- %s: +%s -%s", fields[2], fields[0], fields[1])
	}
}

func renderClassSet(m *markdown, c ClassSetChange) {
	if c.Empty() {
		return
	}
	m.section(headingClassSet)
	for _, name := range c.Added {
		m.line("- added: %s", name)
	}
	for _, name := range c.Removed {
		m.line("- removed: %s", name)
	}
}

func renderClasses(m *markdown, classes map[string]ClassDelta) {
	for _, name := range sortedKeys(classes) {
		d := classes[name]
		if !d.moved() {
			continue
		}
		m.section(headingClassPrefix + name)
		for _, key := range sortedKeys(d.Counts) {
			m.line("- %s: %d -> %d", key, d.Counts[key][0], d.Counts[key][1])
		}
		for _, w := range d.Worsened {
			m.line("- worse: %s", describeRowChange(w))
		}
		for _, n := range d.NewUndisposed {
			m.line("- new trainable without a disposition: %s", n)
		}
	}
}

func describeRowChange(w RowChange) string {
	k := w.Key
	origin := "was " + w.Before.Verdict + ", " + w.Before.Damage
	if w.Added {
		origin = "new row"
	}
	text := fmt.Sprintf("%s %s level %d spell %d rank %d: %s, %s (%s)",
		k.Section, k.Spec, k.Level, k.SpellID, k.Rank, w.After.Verdict, w.After.Damage, origin)
	if w.After.Diff != "" {
		text += "; diff: " + w.After.Diff
	}
	return text
}

func renderTalents(m *markdown, d TalentDelta) {
	if d.Empty() {
		return
	}
	m.section(headingTalents)
	for _, group := range []struct {
		label   string
		changes []TalentChange
	}{{"added", d.Added}, {"removed", d.Removed}, {"changed", d.Changed}} {
		for _, c := range group.changes {
			m.line("- %s %s node %d %s", group.label, c.Class, c.NodeID, c.Name)
			for _, line := range c.Lines {
				m.line("  - %s", line)
			}
		}
	}
}

func renderRacials(m *markdown, changes []RacialChange) {
	if len(changes) == 0 {
		return
	}
	m.section(headingRacials)
	sorted := append([]RacialChange(nil), changes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Race < sorted[j].Race })
	for _, c := range sorted {
		for _, text := range c.Added {
			m.line("- %s added: %s", c.Race, text)
		}
		for _, text := range c.Removed {
			m.line("- %s removed: %s", c.Race, text)
		}
	}
}
