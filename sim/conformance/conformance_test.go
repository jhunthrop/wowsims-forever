package conformance

import (
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

	for _, classSlug := range presetOrder() {
		classSlug := classSlug
		presets := presetsByClass()[classSlug]
		t.Run(classSlug, func(t *testing.T) {
			clientClass, ok := loadClientClass(t, classSlug)
			if !ok {
				return
			}
			trainables := loadTrainables(t, classSlug)

			content, buildErrors := classReport(classSlug, clientClass, trainables, presets)

			if update {
				if err := writeGolden(classSlug, content); err != nil {
					t.Fatalf("writing golden: %v", err)
				}
				t.Logf("wrote %s: %d build errors", goldenPath(classSlug), len(buildErrors))
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

// TestConformanceGoldenIsDeterministic guards the failure mode behind
// the "hunter/mage/warrior/druid/rogue goldens randomly report stale"
// symptom: several spec/level pairs register more than one spell
// sharing a name and rank (Judgement's seal variants, Arcane Missiles'
// eight tick-count ranks — see sortRows' doc comment), so a sort that
// does not fully order ties can render the same underlying rows into
// different bytes from one run to the next. It runs classReport twice,
// independently, for every class with client constants checked in, and
// fails on the first byte where the two runs disagree — a stricter,
// same-process signal than comparing against a committed golden, since
// it cannot be masked by both runs sharing one process's map-iteration
// seed.
func TestConformanceGoldenIsDeterministic(t *testing.T) {
	for _, classSlug := range presetOrder() {
		classSlug := classSlug
		presets := presetsByClass()[classSlug]
		t.Run(classSlug, func(t *testing.T) {
			clientClass, ok := loadClientClass(t, classSlug)
			if !ok {
				return
			}

			trainables := loadTrainables(t, classSlug)

			first, _ := classReport(classSlug, clientClass, trainables, presets)
			second, _ := classReport(classSlug, clientClass, trainables, presets)
			if first != second {
				t.Errorf("%s: classReport produced different output across two runs in the same process; sortRows is not fully deterministic", classSlug)
			}
		})
	}
}

// presetsByClass groups Presets by ClientClassSlug, and presetOrder
// gives the class slugs in Presets' own first-seen order, so both
// TestConformance and TestConformanceGoldenIsDeterministic walk classes
// in the same deterministic order without duplicating the grouping.
func presetsByClass() map[string][]Preset {
	classes := map[string][]Preset{}
	for _, p := range Presets {
		classes[p.ClientClassSlug] = append(classes[p.ClientClassSlug], p)
	}
	return classes
}

func presetOrder() []string {
	var order []string
	seen := map[string]bool{}
	for _, p := range Presets {
		if !seen[p.ClientClassSlug] {
			seen[p.ClientClassSlug] = true
			order = append(order, p.ClientClassSlug)
		}
	}
	return order
}

// loadClientClass loads classSlug's client constants, or skips the
// calling subtest (not failing it) when the fixture is absent, exactly
// as TestConformance always has.
func loadClientClass(t *testing.T, classSlug string) (spellconst.Class, bool) {
	t.Helper()
	path := clientPath(classSlug)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no client constants at %s (copy data/builds/<build>/spellconst/%s.json from the forever repo) -- skipped, not failed", path, classSlug)
		return spellconst.Class{}, false
	}
	clientClass, err := spellconst.Load(path)
	if err != nil {
		t.Fatalf("loading %s: %v", path, err)
	}
	return clientClass, true
}

// loadTrainables loads classSlug's trainables file. Unlike the client
// constants it is never skipped: the golden's trainables section needs it,
// so an absent or malformed file fails the test.
func loadTrainables(t *testing.T, classSlug string) ClassTrainables {
	t.Helper()
	trainables, err := loadClassTrainables(trainablesPath(classSlug))
	if err != nil {
		t.Fatalf("%v (copy data/builds/<build>/trainables/%s.json from the forever repo)", err, classSlug)
	}
	return trainables
}

// TestConformanceSummaryBlocks keeps SUMMARY.md's generated blocks (damage
// counts, trainable-ability gaps) equal to what the goldens' own rows give,
// regenerating them under FOREVER_UPDATE_GOLDEN=1 like the goldens.
func TestConformanceSummaryBlocks(t *testing.T) {
	var slugs []string
	counts := map[string]DamageCounts{}
	gaps := map[string]TrainableGaps{}
	for _, classSlug := range presetOrder() {
		clientClass, ok := loadClientClass(t, classSlug)
		if !ok {
			return
		}
		presets := presetsByClass()[classSlug]
		rows, _, _ := collectRows(clientClass, presets)
		gated, _ := collectTalentGatedRows(clientClass, classSlug)
		slugs = append(slugs, classSlug)
		counts[classSlug] = countDamage(append(rows, gated...))
		gaps[classSlug], _ = trainableGaps(loadTrainables(t, classSlug), presets, gated)
	}

	current, err := readSummary()
	if err != nil {
		t.Fatalf("reading SUMMARY.md: %v", err)
	}
	want := spliceBlock(current, damageBlockBegin, damageBlockEnd, renderDamageBlock(slugs, counts))
	want = spliceBlock(want, trainablesBlockBegin, trainablesBlockEnd, renderTrainablesBlock(slugs, gaps))
	if os.Getenv("FOREVER_UPDATE_GOLDEN") == "1" {
		if err := writeSummary(want); err != nil {
			t.Fatalf("writing SUMMARY.md: %v", err)
		}
		return
	}
	if current != want {
		t.Errorf("SUMMARY.md's generated blocks are stale; run FOREVER_UPDATE_GOLDEN=1 go test --tags=with_db ./sim/conformance/ and review the diff")
	}
}
