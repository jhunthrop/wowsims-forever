// Package clientdamagetest checks a class package's clientdamage tables
// against the vendored client spellconst, the single source they must
// agree with. It lives apart from the clientdamage package so that only
// tests import spellconst's loader.
package clientdamagetest

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// Kind picks which of a client spell's effects a table models: the
// effect code and, when non-zero, the aura code that goes with it.
type Kind struct {
	effect int32
	aura   int32
}

var (
	// Direct is the spell's school-damage effect (effect 2).
	Direct = Kind{effect: 2}
	// Periodic is the spell's periodic-damage aura (effect 6, aura 3);
	// its Amount is per tick.
	Periodic = Kind{effect: 6, aura: 3}
	// PeriodicLeech is a periodic health-leech aura (effect 6, aura 53),
	// the drains: Amount is per tick.
	PeriodicLeech = Kind{effect: 6, aura: 53}
	// HealthLeech is a direct health-leech effect (effect 9), Death Coil.
	HealthLeech = Kind{effect: 9}
	// NormalizedWeapon is the flat bonus of a normalized-weapon-damage
	// effect (effect 121), the part that is not the weapon.
	NormalizedWeapon = Kind{effect: 121}
)

const (
	// rollTolerance is how far a rolled bound may sit from the client's
	// float: the tables round the client's variance and growth to six
	// decimals.
	rollTolerance = 0.05
	// coefficientTolerance covers the client's float32 coefficients.
	coefficientTolerance = 0.002
)

// checkLevels are the caster levels every rank is compared at: the
// presets the conformance report builds, so the per-level growth and its
// cap are both exercised.
var checkLevels = []int{1, 10, 20, 30, 38, 40, 50, 60}

// Load reads one class's vendored client spellconst. path is relative to
// the calling test's package directory.
func Load(t *testing.T, path string) spellconst.Class {
	t.Helper()
	class, err := spellconst.Load(path)
	if err != nil {
		t.Fatalf("loading the client's spellconst %s: %v", path, err)
	}
	return class
}

// clientEffect is the effect of the given kind on a client spell.
func clientEffect(spell spellconst.Spell, kind Kind) (spellconst.Effect, bool) {
	for _, effect := range spell.Effects {
		if effect.Effect == kind.effect && (kind.aura == 0 || effect.Aura == kind.aura) {
			return effect, true
		}
	}
	return spellconst.Effect{}, false
}

// AssertTable checks every rank of one spell's table against the client:
// the roll at each checked level, and, when coefficients is non-nil, the
// spell-power coefficient. Index 0 of every slice is the unused rank 0.
func AssertTable(t *testing.T, class spellconst.Class, kind Kind, name string, ids []int32, effects []clientdamage.Effect, coefficients []float64) {
	t.Helper()
	if len(ids) != len(effects) {
		t.Fatalf("%s: %d spell ids but %d damage effects", name, len(ids), len(effects))
	}
	for rank := 1; rank < len(ids); rank++ {
		spell, ok := class.ByID(ids[rank])
		if !ok {
			t.Fatalf("%s rank %d: spell %d is not in the client's spellconst", name, rank, ids[rank])
		}
		client, ok := clientEffect(spell, kind)
		if !ok {
			t.Fatalf("%s rank %d (%d): the client spell has no effect %d (aura %d)", name, rank, ids[rank], kind.effect, kind.aura)
		}
		assertRolls(t, name, rank, spell, client, effects[rank])
		if coefficients != nil && math.Abs(coefficients[rank]-client.ResolvedSPCoefficient) > coefficientTolerance {
			t.Errorf("%s rank %d (%d): engine coefficient %v, client %v", name, rank, ids[rank], coefficients[rank], client.ResolvedSPCoefficient)
		}
	}
}

func assertRolls(t *testing.T, name string, rank int, spell spellconst.Spell, client spellconst.Effect, got clientdamage.Effect) {
	t.Helper()
	if got.SpellLevel != spell.SpellLevel || got.MaxLevel != spell.MaxLevel {
		t.Errorf("%s rank %d: engine spell level %d / max level %d, client %d / %d", name, rank, got.SpellLevel, got.MaxLevel, spell.SpellLevel, spell.MaxLevel)
	}
	for _, level := range checkLevels {
		wantMin, wantMax, _ := spell.DamageRange(client.Index, level)
		have := got.Range(level)
		if math.Abs(have[0]-wantMin) > rollTolerance || math.Abs(have[1]-wantMax) > rollTolerance {
			t.Errorf("%s rank %d at level %d: engine rolls %.2f-%.2f, client %.2f-%.2f", name, rank, level, have[0], have[1], wantMin, wantMax)
		}
	}
}
