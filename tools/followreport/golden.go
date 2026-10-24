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
	colDamage            = "Damage"
	colAbility           = "Ability"
	colDisposition       = "Disposition"
	dispositionNone      = ""
	dispositionUnsorted  = "unclassified"
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
type Row struct{ Verdict, Damage string }

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

// parseTable reads the first markdown table in the section's lines.
func parseTable(lines []string) (table, bool) {
	var t table
	started := false
	for _, line := range lines {
		if !strings.HasPrefix(line, tableRowPrefix) {
			if started {
				break
			}
			continue
		}
		cells := splitCells(line)
		switch {
		case !started:
			t.header, started = cells, true
		case strings.HasPrefix(cells[0], "---"):
		default:
			t.rows = append(t.rows, cells)
		}
	}
	return t, started
}

// splitSections maps each "## " heading line to the lines beneath it.
func splitSections(lines []string) map[string][]string {
	sections := map[string][]string{}
	current := ""
	for _, line := range lines {
		if strings.HasPrefix(line, headingPrefix) {
			current = strings.TrimSpace(line)
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

func parseSpellRows(section string, lines []string, into map[RowKey]Row) error {
	t, ok := parseTable(lines)
	if !ok {
		return nil
	}
	idx, err := t.requireColumns(colSpec, colLevel, colRank, colSpellID, colVerdict, colDamage)
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
		into[key] = Row{Verdict: cells[idx[4]], Damage: cells[idx[5]]}
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
	t, ok := parseTable(lines)
	if !ok {
		return out, nil
	}
	abilityCol := t.column(colAbility)
	if abilityCol < 0 {
		return nil, fmt.Errorf("trainables: no %q column in header %v", colAbility, t.header)
	}
	dispCol := t.column(colDisposition)
	for _, cells := range t.rows {
		if len(cells) != len(t.header) {
			return nil, fmt.Errorf("trainables: row has %d cells, header has %d: %v", len(cells), len(t.header), cells)
		}
		tr := Trainable{Name: cells[abilityCol], HasDispositionColumn: dispCol >= 0}
		if dispCol >= 0 {
			tr.Disposition = cells[dispCol]
		}
		out[tr.Name] = tr
	}
	return out, nil
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
		if err := parseSpellRows(section, sections[heading], g.Rows); err != nil {
			return Golden{}, err
		}
	}
	if g.Trainables, err = parseTrainables(sections[headingTrainables]); err != nil {
		return Golden{}, err
	}
	return g, nil
}
