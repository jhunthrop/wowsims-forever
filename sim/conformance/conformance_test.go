package conformance

import (
	"fmt"
	"os"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

func init() {
	registerAll()
}

// TestConformance is Phase 1b of the rotation accuracy program
// (docs/superpowers/specs/2026-09-28-rotation-accuracy-program-design.md):
// for every registered spec, at every level in Levels, every spell with a
// real ActionID.SpellID is checked against the Forever client's own spell
// constants, and the result is written to
// sim/core/testdata/conformance/<class>.golden.md. It fails only when a
// class's golden differs from what is committed
// (FOREVER_UPDATE_GOLDEN=1 regenerates and writes instead of failing), so
// the first run records today's gaps and every later change is judged
// against them.
func TestConformance(t *testing.T) {
	update := os.Getenv("FOREVER_UPDATE_GOLDEN") == "1"

	classes := map[string][]Preset{}
	var order []string
	for _, p := range Presets {
		if _, seen := classes[p.ClientClassSlug]; !seen {
			order = append(order, p.ClientClassSlug)
		}
		classes[p.ClientClassSlug] = append(classes[p.ClientClassSlug], p)
	}

	for _, classSlug := range order {
		classSlug := classSlug
		presets := classes[classSlug]
		t.Run(classSlug, func(t *testing.T) {
			path := clientPath(classSlug)
			if _, err := os.Stat(path); err != nil {
				t.Skipf("no client constants at %s (copy data/builds/<build>/spellconst/%s.json from the forever repo) -- skipped, not failed", path, classSlug)
				return
			}
			clientClass, err := spellconst.Load(path)
			if err != nil {
				t.Fatalf("loading %s: %v", path, err)
			}

			var rows []Row
			var buildErrors []string
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
			content := renderGolden(classSlug, clientClass.Build, rows, buildErrors)

			if update {
				if err := writeGolden(classSlug, content); err != nil {
					t.Fatalf("writing golden: %v", err)
				}
				t.Logf("wrote %s: %d rows, %d build errors", goldenPath(classSlug), len(rows), len(buildErrors))
				return
			}

			committed, err := readGolden(classSlug)
			if err != nil {
				t.Fatalf("%s has no committed golden yet; run FOREVER_UPDATE_GOLDEN=1 go test ./sim/conformance/... and commit it: %v", goldenPath(classSlug), err)
			}
			if committed != content {
				t.Errorf("%s is stale; run FOREVER_UPDATE_GOLDEN=1 go test ./sim/conformance/... and review the diff before committing", goldenPath(classSlug))
			}
		})
	}
}
