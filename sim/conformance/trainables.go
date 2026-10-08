package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TrainablesDir is where this package's copy of the client's per-class
// trainable abilities lives (sim/core/testdata/conformance/client/trainables/
// <slug>.json, copied from the site's data/builds/<build>/trainables/<slug>.json,
// the same way ClientDir's spellconst files are). It is a directory of its own
// because spellconst's pipeline globs *.json beside the spellconst files.
const TrainablesDir = "../core/testdata/conformance/client/trainables"

// Trainable is one ability a class learns, with every rank the client lists,
// as the site's pipeline.trainables writes it. Only the fields this report
// reads are decoded; the loader rejects any field it does not know, so a
// schema change in the pipeline fails here instead of being silently ignored.
type Trainable struct {
	Name       string          `json:"name"`
	SkillLine  string          `json:"skill_line"`
	Source     string          `json:"source"`
	Active     bool            `json:"active"`
	Cost       float64         `json:"cost"`
	CostType   int32           `json:"cost_type"`
	CastTimeMS int32           `json:"cast_time_ms"`
	CooldownMS int32           `json:"cooldown_ms"`
	Ranks      []TrainableRank `json:"ranks"`
}

// TrainableRank is one rank of a Trainable: a spell id, its rank number
// (0 for an unranked spell) and the level the client teaches it at.
type TrainableRank struct {
	ID    int32 `json:"id"`
	Rank  int   `json:"rank"`
	Level int   `json:"level"`
}

// ClassTrainables is one class's trainables file.
type ClassTrainables struct {
	Build      string      `json:"build"`
	ClassSlug  string      `json:"class_slug"`
	Trainables []Trainable `json:"trainables"`
}

func trainablesPath(classSlug string) string {
	return filepath.Join(TrainablesDir, classSlug+".json")
}

// loadClassTrainables reads one class's trainables file.
func loadClassTrainables(path string) (ClassTrainables, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ClassTrainables{}, fmt.Errorf("trainables: %w", err)
	}
	var c ClassTrainables
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return ClassTrainables{}, fmt.Errorf("trainables: parsing %s: %w", path, err)
	}
	if c.ClassSlug == "" || c.Build == "" {
		return ClassTrainables{}, fmt.Errorf("trainables: %s needs class_slug and build", path)
	}
	return c, nil
}

// firstLevel is the level the ability's first rank is learned at.
func (t Trainable) firstLevel() int {
	first := 0
	for i, r := range t.Ranks {
		if i == 0 || r.Level < first {
			first = r.Level
		}
	}
	return first
}

// registeredIn reports whether the engine registers any rank of t.
func (t Trainable) registeredIn(registered map[int32]bool) bool {
	for _, r := range t.Ranks {
		if registered[r.ID] {
			return true
		}
	}
	return false
}

// whyItMatters names what makes the ability one a rotation could press: a
// power cost, a cast time, a cooldown. Every active trainable has at least one.
func (t Trainable) whyItMatters() string {
	var reasons []string
	if t.Cost > 0 {
		reasons = append(reasons, "power cost")
	}
	if t.CastTimeMS > 0 {
		reasons = append(reasons, "cast time")
	}
	if t.CooldownMS > 0 {
		reasons = append(reasons, "cooldown")
	}
	return strings.Join(reasons, ", ")
}

// TrainableGaps is one class's comparison of its trainables against the
// engine's registered spells.
type TrainableGaps struct {
	// Active is the number of active trainables the class has.
	Active int
	// Unregistered lists the active trainables with a client learn row
	// (source skill_line_ability) the engine registers no rank of, ordered
	// by first level, then name.
	Unregistered []Trainable
	// NoLearnRow lists the active class-family spells the client lists on no
	// learn row (source class_spell) that the engine does not register. They
	// are not counted in Active or Unregistered.
	NoLearnRow []Trainable
}

// sourceClassSpell is the pipeline's tag for a spell found through its class
// family rather than a SkillLineAbility learn row.
const sourceClassSpell = "class_spell"

// registeredSpellIDs is every spell id the class's presets register in their
// spellbook at any level in Levels (empty-talent builds), plus every id a
// talent-gated build registers: an ability is "registered" when the engine can
// cast it in some spec at some level.
func registeredSpellIDs(presets []Preset, gatedRows []Row) (ids map[int32]bool, buildErrors []string) {
	ids = map[int32]bool{}
	for _, preset := range presets {
		for _, level := range Levels {
			built, _, err := buildForComparison(preset, level)
			if err != nil {
				buildErrors = append(buildErrors, fmt.Sprintf("%s/L%d: registered-spell scan: %v", preset.Label, level, err))
				continue
			}
			for _, spell := range built.Spellbook {
				ids[spell.ActionID.SpellID] = true
			}
		}
	}
	for _, row := range gatedRows {
		ids[row.SpellID] = true
	}
	return ids, buildErrors
}

// compareTrainables splits a class's active trainables into those the engine
// registers and those it does not.
func compareTrainables(trainables ClassTrainables, registered map[int32]bool) TrainableGaps {
	var gaps TrainableGaps
	for _, t := range trainables.Trainables {
		if !t.Active {
			continue
		}
		if t.Source == sourceClassSpell {
			if !t.registeredIn(registered) {
				gaps.NoLearnRow = append(gaps.NoLearnRow, t)
			}
			continue
		}
		gaps.Active++
		if !t.registeredIn(registered) {
			gaps.Unregistered = append(gaps.Unregistered, t)
		}
	}
	sortTrainables(gaps.Unregistered)
	sortTrainables(gaps.NoLearnRow)
	return gaps
}

func sortTrainables(list []Trainable) {
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.firstLevel() != b.firstLevel() {
			return a.firstLevel() < b.firstLevel()
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Ranks[0].ID < b.Ranks[0].ID
	})
}

// trainableGaps runs the comparison for one class end to end.
func trainableGaps(trainables ClassTrainables, presets []Preset, gatedRows []Row) (TrainableGaps, []string) {
	registered, buildErrors := registeredSpellIDs(presets, gatedRows)
	return compareTrainables(trainables, registered), buildErrors
}

// renderTrainableGaps writes the "Trainable abilities the engine does not
// register" section and its "In the client, no learn row" subsection.
func renderTrainableGaps(b *strings.Builder, classSlug string, gaps TrainableGaps) {
	fmt.Fprintf(b, "## Trainable abilities the engine does not register\n\n")
	fmt.Fprintf(b, "Active trainables (pipeline.trainables: SkillLineAbility rows with AcquireMethod 0 and a learn level above 0 on the class skill lines, so Season of Discovery runes are excluded; active means a power cost, a cast time or a cooldown) for which no rank's spell id appears in any spec's spellbook at any level in this report, nor in a talent-gated build. %d of the class's %d active trainables are listed. This report only compares the spells the engine declares, so these are invisible to the tables above. Utility spells (Polymorph, Blink, teleports) are expected here; the Why column says what a rotation would care about. Cost is in the client's units (rage in tenths).\n\n", len(gaps.Unregistered), gaps.Active)
	renderTrainableTable(b, classSlug, gaps.Unregistered)

	fmt.Fprintf(b, "### In the client, no learn row\n\n")
	fmt.Fprintf(b, "Active, ranked, levelled class-family spells the client lists on no SkillLineAbility row (Unstable Affliction, Hydra Shot) that the engine does not register. They are not counted above or in SUMMARY.md; the list also carries spells that are probably not player spellbook entries (rogue poisons, NPC volleys).\n\n")
	renderTrainableTable(b, classSlug, gaps.NoLearnRow)
}

func renderTrainableTable(b *strings.Builder, classSlug string, list []Trainable) {
	if len(list) == 0 {
		fmt.Fprintf(b, "None.\n\n")
		return
	}
	dispositions, classified := trainableDispositions[classSlug]
	header, rule := "| Ability | Level (first→last) | Ranks | Skill line | Source | Cost | Cast ms | Cooldown ms | Why it matters |", "|---|---|---|---|---|---|---|---|---|"
	if classified {
		header, rule = strings.TrimSuffix(header, "|")+"| Disposition |", rule+"---|"
	}
	fmt.Fprintf(b, "%s\n%s\n", header, rule)
	for _, t := range list {
		last := t.Ranks[len(t.Ranks)-1]
		cost := "0"
		if t.Cost > 0 {
			cost = fmt.Sprintf("%.0f %s", t.Cost, clientCostTypeName(t.CostType))
		}
		skillLine := t.SkillLine
		if skillLine == "" {
			skillLine = "n/a"
		}
		disposition := ""
		if classified {
			disposition = " " + dispositionFor(dispositions, t.Name) + " |"
		}
		fmt.Fprintf(b, "| %s (%d) | %d→%d | %d | %s | %s | %s | %d | %d | %s |%s\n",
			t.Name, t.Ranks[0].ID, t.firstLevel(), last.Level, len(t.Ranks), skillLine, t.Source,
			cost, t.CastTimeMS, t.CooldownMS, t.whyItMatters(), disposition)
	}
	fmt.Fprintf(b, "\n")
}

// unclassifiedDisposition is what a classified class's table prints for an
// ability with no entry; a test fails on it.
const unclassifiedDisposition = "unclassified"

func dispositionFor(dispositions map[string]string, name string) string {
	if d, ok := dispositions[name]; ok {
		return d
	}
	return unclassifiedDisposition
}
