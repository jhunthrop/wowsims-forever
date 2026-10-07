package conformance

import (
	"fmt"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// classReport runs one class's full conformance pipeline: build every
// preset at every level with an empty talent string, compare each
// registered spell against the client, collect the class's
// talent-gated spells (TalentGatedSpells) as their own section, sort
// both, and render the golden markdown. It is the single code path
// TestConformance and TestConformanceGoldenIsDeterministic both call,
// so the committed golden and the determinism check can never drift
// out of sync with each other.
func classReport(classSlug string, clientClass spellconst.Class, presets []Preset) (content string, buildErrors []string) {
	rows, notes, rowErrors := collectRows(clientClass, presets)
	talentGatedRows, talentGatedErrors := collectTalentGatedRows(clientClass, classSlug)

	buildErrors = append(append([]string{}, rowErrors...), talentGatedErrors...)
	content = renderGolden(classSlug, clientClass.Build, rows, talentGatedRows, notes, buildErrors)
	return content, buildErrors
}

// collectRows builds every preset at every level in Levels - with an
// empty talent string, falling back to the preset's real one only if
// that fails (buildForComparison) - and compares each spell in its
// Spellbook against the client's constants, deduping on (spec, level,
// spellID, rank) exactly as the golden's own row-per-registration shape
// requires, then hands the result to sortRows for a deterministic
// order. A build error for one preset/level does not stop the rest from
// being collected. notes carries every fallback buildForComparison had
// to take, for the golden header to record.
func collectRows(clientClass spellconst.Class, presets []Preset) (rows []Row, notes []string, buildErrors []string) {
	seen := map[string]bool{}
	for _, preset := range presets {
		for _, level := range Levels {
			built, note, err := buildForComparison(preset, level)
			if err != nil {
				buildErrors = append(buildErrors, fmt.Sprintf("%s/L%d: %v", preset.Label, level, err))
				continue
			}
			if note != "" {
				notes = append(notes, note)
			}
			for _, spell := range built.Spellbook {
				row, ok := rowFor(clientClass, preset, level, built, spell)
				if !ok {
					continue
				}
				key := fmt.Sprintf("%s|%d|%d|%d", row.Spec, row.Level, row.SpellID, row.Rank)
				if seen[key] {
					continue
				}
				seen[key] = true
				rows = append(rows, row)
			}
		}
	}
	sortRows(rows)
	return rows, notes, buildErrors
}
