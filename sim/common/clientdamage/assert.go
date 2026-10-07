package clientdamage

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// rollTolerance is how far a registered roll may sit from the client's:
// the generated tables are rounded to four decimals.
const rollTolerance = 0.01

// LoadClient loads the vendored copy of the site's spellconst/<class>.json
// (the file the conformance report reads) for a class package's tests;
// relativeDir is the path from the test's package to sim/.
func LoadClient(t *testing.T, relativeDir, class string) spellconst.Class {
	t.Helper()
	loaded, err := spellconst.Load(relativeDir + "/core/testdata/conformance/client/" + class + ".json")
	if err != nil {
		t.Fatalf("loading the client's %s spellconst: %v", class, err)
	}
	return loaded
}

// AssertRoll fails unless got is the roll the client states for the
// given effect of spellID at casterLevel (spellconst.Spell.DamageRange).
func AssertRoll(t *testing.T, client spellconst.Class, label string, spellID int32, effectIndex, casterLevel int, got [2]float64) {
	t.Helper()
	spell, ok := client.ByID(spellID)
	if !ok {
		t.Fatalf("%s: spell %d is not in the client table", label, spellID)
	}
	low, high, ok := spell.DamageRange(effectIndex, casterLevel)
	if !ok {
		t.Fatalf("%s: spell %d has no effect %d", label, spellID, effectIndex)
	}
	if math.Abs(got[0]-low) > rollTolerance || math.Abs(got[1]-high) > rollTolerance {
		t.Errorf("%s (%d) at level %d: engine rolls %.4f-%.4f, client %.4f-%.4f", label, spellID, casterLevel, got[0], got[1], low, high)
	}
}
