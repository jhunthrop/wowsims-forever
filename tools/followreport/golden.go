package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Section headings of a conformance golden, and the section names RowKey uses.
const (
	headingSpells        = "## Spells"
	headingTalentGated   = "## Talent-gated spells"
	headingTrainables    = "## Trainable abilities the engine does not register"
	sectionSpells        = "spells"
	sectionTalentGated   = "talent-gated"
	h1Prefix             = "# "
	tableRowPrefix       = "|"
	escapedPipe          = `\|`
	pipePlaceholder      = "\x00"
	headingPrefix        = "## "
	colSpec              = "Spec"
	colLevel             = "Level"
	colSpell             = "Spell"
	colRank              = "Rank"
	colSpellID           = "SpellID"
	colVerdict           = "Verdict"
	colDiff              = "Diff"
	colDamage            = "Damage"
	colAbility           = "Ability"
	colDisposition       = "Disposition"
	dispositionNone      = ""
	dispositionUnsorted  = "unclassified"
	verdictMatch         = "match"
	verdictMismatch      = "mismatch"
	damageDeclaredDiffer = "declared, differs"
)

// RowKey identifies one spell row of a golden across two builds.
type RowKey struct {
	Section, Spec string
	Level         int
	SpellID       int
	Rank          int
}

// Row is the part of a spell row the verdict reads.
type Row struct{ Verdict, Damage, Diff string }

// Trainable is one row of the unregistered-trainables table.
type Trainable struct {
	Name, Disposition    string
	HasDispositionColumn bool
}

// Golden is one class's parsed conformance golden.
type Golden struct {
	Class      string
	Rows       map[RowKey]Row
	Trainables map[string]Trainable
}

type table struct {
	header []string
	rows   [][]string
}

// column returns the index of the header cell with exactly this text, or -1.
func (t table) column(name string) int {
	for i, h := range t.header {
		if h == name {
			return i
		}
	}
	return -1
}

// requireColumns returns the indexes of the named columns, in order.
func (t table) requireColumns(names ...string) ([]int, error) {
	idx := make([]int, len(names))
	for i, name := range names {
		if idx[i] = t.column(name); idx[i] < 0 {
			return nil, fmt.Errorf("no %q column in header %v", name, t.header)
		}
	}
	return idx, nil
}

func splitCells(line string) []string {
	line = strings.ReplaceAll(line, escapedPipe, pipePlaceholder)
	line = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(line), tableRowPrefix), tableRowPrefix)
	cells := strings.Split(line, "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(strings.ReplaceAll(c, pipePlaceholder, "|"))
	}
	return cells
}

// parseTables reads every markdown table in the section's lines, in order.
func parseTables(lines []string) []table {
	var tables []table
	inTable := false
	for _, line := range lines {
		if !strings.HasPrefix(line, tableRowPrefix) {
			inTable = false
			continue
		}
		cells := splitCells(line)
		switch {
		case !inTable:
			tables, inTable = append(tables, table{header: cells}), true
		case strings.HasPrefix(cells[0], "---"):
		default:
			last := &tables[len(tables)-1]
			last.rows = append(last.rows, cells)
		}
	}
	return tables
}

// splitSections maps each "## " heading line to the lines beneath it.
func splitSections(lines []string) map[string][]string {
	sections := map[string][]string{}
	current := ""
	for _, line := range lines {
		if strings.HasPrefix(line, headingPrefix) {
			current = strings.TrimSpace(line)
			sections[current] = nil
			continue
		}
		if current != "" {
			sections[current] = append(sections[current], line)
		}
	}
	return sections
}

func className(lines []string) (string, error) {
	for _, line := range lines {
		if rest, ok := strings.CutPrefix(line, h1Prefix); ok {
			name, _, _ := strings.Cut(rest, ":")
			return strings.TrimSpace(name), nil
		}
	}
	return "", fmt.Errorf("no H1 class heading")
}

func parseSpellRows(section, heading string, lines []string, present bool, into map[RowKey]Row) error {
	if !present {
		return fmt.Errorf("no %q section", heading)
	}
	for _, t := range parseTables(lines) {
		if err := addSpellRows(section, t, into); err != nil {
			return err
		}
	}
	return nil
}

func addSpellRows(section string, t table, into map[RowKey]Row) error {
	idx, err := t.requireColumns(colSpec, colLevel, colRank, colSpellID, colVerdict, colDamage, colDiff)
	if err != nil {
		return fmt.Errorf("%s: %w", section, err)
	}
	for _, cells := range t.rows {
		if len(cells) != len(t.header) {
			return fmt.Errorf("%s: row has %d cells, header has %d: %v", section, len(cells), len(t.header), cells)
		}
		key, err := rowKey(section, cells, idx)
		if err != nil {
			return err
		}
		if _, dup := into[key]; dup {
			return fmt.Errorf("%s: duplicate row %+v", section, key)
		}
		into[key] = Row{Verdict: cells[idx[4]], Damage: cells[idx[5]], Diff: cells[idx[6]]}
	}
	return nil
}

func rowKey(section string, cells []string, idx []int) (RowKey, error) {
	nums := make([]int, 3)
	for i, col := range []int{idx[1], idx[2], idx[3]} {
		n, err := strconv.Atoi(cells[col])
		if err != nil {
			return RowKey{}, fmt.Errorf("%s: cell %q is not a number: %w", section, cells[col], err)
		}
		nums[i] = n
	}
	return RowKey{Section: section, Spec: cells[idx[0]], Level: nums[0], Rank: nums[1], SpellID: nums[2]}, nil
}

func parseTrainables(lines []string) (map[string]Trainable, error) {
	out := map[string]Trainable{}
	for _, t := range parseTables(lines) {
		if err := addTrainables(t, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func addTrainables(t table, into map[string]Trainable) error {
	abilityCol := t.column(colAbility)
	if abilityCol < 0 {
		return fmt.Errorf("trainables: no %q column in header %v", colAbility, t.header)
	}
	dispCol := t.column(colDisposition)
	for _, cells := range t.rows {
		if len(cells) != len(t.header) {
			return fmt.Errorf("trainables: row has %d cells, header has %d: %v", len(cells), len(t.header), cells)
		}
		tr := Trainable{Name: cells[abilityCol], HasDispositionColumn: dispCol >= 0}
		if dispCol >= 0 {
			tr.Disposition = cells[dispCol]
		}
		if _, dup := into[tr.Name]; dup {
			return fmt.Errorf("trainables: duplicate ability %q", tr.Name)
		}
		into[tr.Name] = tr
	}
	return nil
}

// ParseGolden reads one class's conformance golden.
func ParseGolden(path string) (Golden, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Golden{}, fmt.Errorf("followreport: %s: %w", path, err)
	}
	g, err := parseGoldenText(string(body))
	if err != nil {
		return Golden{}, fmt.Errorf("followreport: %s: %w", path, err)
	}
	return g, nil
}

func parseGoldenText(text string) (Golden, error) {
	lines := strings.Split(text, "\n")
	class, err := className(lines)
	if err != nil {
		return Golden{}, err
	}
	g := Golden{Class: class, Rows: map[RowKey]Row{}}
	sections := splitSections(lines)
	for heading, section := range map[string]string{headingSpells: sectionSpells, headingTalentGated: sectionTalentGated} {
		lines, present := sections[heading]
		if err := parseSpellRows(section, heading, lines, present, g.Rows); err != nil {
			return Golden{}, err
		}
	}
	if g.Trainables, err = parseTrainables(sections[headingTrainables]); err != nil {
		return Golden{}, err
	}
	return g, nil
}
