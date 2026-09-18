// Package spellconst reads the per-class spell constants the data
// pipeline generates from the client tables, so an ability's numbers are
// regenerated rather than retyped when Forever changes one.
//
// The pipeline emits the DB2 columns verbatim. For Classic-lineage spells
// EffectBonusCoefficient is routinely 0 or wrong, so a zero here means
// "the table does not know", and CoefficientFor supplies the vanilla
// convention in its place. The conventions live in this package and
// nowhere else.
package spellconst

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// gcd is the vanilla global cooldown. A cast faster than it still scales
// as a GCD cast, which is why an instant nuke is not coefficient zero.
const gcd = 1500 * time.Millisecond

// directDivisor and periodicDivisor are the vanilla spell-coefficient
// conventions: a direct spell gets cast_time/3.5, a periodic one
// duration/15. Both are seconds.
const (
	directDivisor   = 3.5
	periodicDivisor = 15.0
)

// Spell is one rank of one ability.
type Spell struct {
	ID             int32   `json:"id"`
	Name           string  `json:"name"`
	Rank           int     `json:"rank"`
	BasePointsLow  float64 `json:"base_points_low"`
	BasePointsHigh float64 `json:"base_points_high"`
	// Coefficient is the spell-power coefficient, resolved: the table's
	// value when it has one, the convention's when it does not.
	Coefficient float64 `json:"coefficient"`
	// CoefficientSource is "table" or "convention", so a reader can tell
	// a measured number from a derived one.
	CoefficientSource string  `json:"-"`
	CooldownMS        int32   `json:"cooldown_ms"`
	CastTimeMS        int32   `json:"cast_time_ms"`
	DurationMS        int32   `json:"duration_ms"`
	Cost              float64 `json:"cost"`
	School            int32   `json:"school"`
	FamilyMask        uint64  `json:"family_mask"`
	Level             int     `json:"level"`
}

// Class is one generated per-class file.
type Class struct {
	Slug   string  `json:"class_slug"`
	Build  string  `json:"build"`
	Spells []Spell `json:"spells"`

	// hybrid marks the classes whose spell coefficients are halved by the
	// vanilla convention.
	hybrid bool
}

// hybridClasses are the classes the vanilla convention halves.
var hybridClasses = map[string]bool{
	"paladin": true,
	"shaman":  true,
	"druid":   true,
	"priest":  false, // shadow priests use the full convention
	"warrior": false,
	"rogue":   false,
	"hunter":  false,
	"mage":    false,
	"warlock": false,
}

// Load reads a generated class file and resolves every coefficient.
func Load(path string) (Class, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Class{}, fmt.Errorf("spellconst: %w", err)
	}
	var c Class
	if err := json.Unmarshal(b, &c); err != nil {
		return Class{}, fmt.Errorf("spellconst: parsing %s: %w", path, err)
	}
	if c.Slug == "" {
		return Class{}, fmt.Errorf("spellconst: %s has no class_slug", path)
	}
	if c.Build == "" {
		return Class{}, fmt.Errorf("spellconst: %s has no build; a constants file must record which client it came from", path)
	}
	c.hybrid = hybridClasses[c.Slug]
	for i := range c.Spells {
		s := &c.Spells[i]
		if s.Coefficient != 0 {
			s.CoefficientSource = "table"
			continue
		}
		s.Coefficient, s.CoefficientSource = CoefficientFor(s.CastTimeMS, s.DurationMS, c.hybrid)
	}
	sort.SliceStable(c.Spells, func(i, j int) bool {
		if c.Spells[i].Name != c.Spells[j].Name {
			return c.Spells[i].Name < c.Spells[j].Name
		}
		return c.Spells[i].Rank < c.Spells[j].Rank
	})
	return c, nil
}

// ByID returns one rank of one spell.
func (c Class) ByID(id int32) (Spell, bool) {
	for _, s := range c.Spells {
		if s.ID == id {
			return s, true
		}
	}
	return Spell{}, false
}

// Ranks returns every rank of a named spell, ascending, or nil.
func (c Class) Ranks(name string) []Spell {
	var out []Spell
	for _, s := range c.Spells {
		if s.Name == name {
			out = append(out, s)
		}
	}
	return out
}

// CoefficientFor is the vanilla spell-coefficient convention: a direct
// spell scales with its cast time over 3.5 seconds, a periodic one with
// its duration over 15, and a hybrid class gets half. A cast faster than
// the global cooldown is treated as a GCD cast.
//
// This is a convention, not data: per-spell exceptions are dozens strong
// and live in the ability files that override this value, exactly as they
// do today. The second return says which of the two a caller got, so an
// override can be applied knowingly.
func CoefficientFor(castTimeMS int32, durationMS int32, hybrid bool) (float64, string) {
	var coeff float64
	switch {
	case durationMS > 0:
		coeff = (time.Duration(durationMS) * time.Millisecond).Seconds() / periodicDivisor
	default:
		cast := time.Duration(castTimeMS) * time.Millisecond
		if cast < gcd {
			cast = gcd
		}
		coeff = cast.Seconds() / directDivisor
	}
	if hybrid {
		coeff /= 2
	}
	return coeff, "convention"
}
