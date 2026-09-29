package conformance

import (
	"fmt"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// classReport runs one class's full conformance pipeline: build every
// preset at every level, compare each registered spell against the
// client, sort, and render the golden markdown. It is the single code
// path TestConformance and TestConformanceGoldenIsDeterministic both
// call, so the committed golden and the determinism check can never
// drift out of sync with each other.
func classReport(classSlug string, clientClass spellconst.Class, presets []Preset) (content string, buildErrors []string) {
	rows, buildErrors := collectRows(clientClass, presets)
	content = renderGolden(classSlug, clientClass.Build, rows, buildErrors)
	return content, buildErrors
}

// collectRows builds every preset at every level in Levels and compares
// each spell in its Spellbook against the client's constants, deduping
// on (spec, level, spellID, rank) exactly as the golden's own row-per-
// registration shape requires, then hands the result to sortRows for a
// deterministic order. A build error for one preset/level does not stop
// the rest from being collected.
func collectRows(clientClass spellconst.Class, presets []Preset) (rows []Row, buildErrors []string) {
	seen := map[string]bool{}
	for _, preset := range presets {
		for _, level := range Levels {
			built, err := buildCharacter(preset, level)
			if err != nil {
				buildErrors = append(buildErrors, fmt.Sprintf("%s/L%d: %v", preset.Label, level, err))
				continue
			}
			for _, spell := range built.Spellbook {
				row, ok := rowFor(clientClass, preset, level, spell)
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
	return rows, buildErrors
}
