// Package spellconst reads the per-class spell constants the data
// pipeline generates from the client tables, so an ability's numbers are
// regenerated rather than retyped when Forever changes one.
//
// The pipeline emits the DB2 columns verbatim, per effect. For
// Classic-lineage spells EffectBonusCoefficient is routinely 0 or wrong,
// so a zero spell-power coefficient here means "the table does not
// know", and CoefficientFor supplies the vanilla convention in its
// place. The conventions live in this package and nowhere else.
//
// Shape: docs/superpowers/specs/2026-09-14-simulator-interfaces.md, the
// `simconst` bullet (amended 2026-09-18 to the shape the data lane
// actually emits: `spells` is an object keyed by spell id, not an
// array, and coefficients live on each effect, not on the spell).
package spellconst

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
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

// Effect is one effect slot of one spell rank, verbatim from the DB2
// columns the pipeline emits.
type Effect struct {
	Index         int     `json:"index"`
	Effect        int32   `json:"effect"`
	Aura          int32   `json:"aura"`
	Amount        float64 `json:"amount"`
	SPCoefficient float64 `json:"sp_coefficient"`
	APCoefficient float64 `json:"ap_coefficient"`
	PeriodMS      int32   `json:"period_ms"`
	MiscValue     int32   `json:"misc_value"`
	TriggerSpell  int32   `json:"trigger_spell"`

	// ResolvedSPCoefficient is SPCoefficient when the table supplied a
	// nonzero value, or the vanilla convention's value when it did not.
	ResolvedSPCoefficient float64 `json:"-"`
	// CoefficientSource is "table" or "convention", so a reader can tell
	// a measured number from a derived one.
	CoefficientSource string `json:"-"`
}

// spellBody is the JSON body of one entry in the `spells` object. The id
// itself is the object's key, not a field in the body.
type spellBody struct {
	Name               string   `json:"name"`
	Rank               int      `json:"rank"`
	SchoolMask         int32    `json:"school_mask"`
	CastTimeMS         int32    `json:"cast_time_ms"`
	GCDMS              int32    `json:"gcd_ms"`
	CooldownMS         int32    `json:"cooldown_ms"`
	CategoryCooldownMS int32    `json:"category_cooldown_ms"`
	DurationMS         int32    `json:"duration_ms"`
	Cost               float64  `json:"cost"`
	CostType           int32    `json:"cost_type"`
	SpellLevel         int      `json:"spell_level"`
	FamilyMask         [4]int64 `json:"family_mask"`
	Effects            []Effect `json:"effects"`
}

// Spell is one rank of one ability, with its id resolved from the
// `spells` object's key.
type Spell struct {
	ID                 int32
	Name               string
	Rank               int
	SchoolMask         int32
	CastTimeMS         int32
	GCDMS              int32
	CooldownMS         int32
	CategoryCooldownMS int32
	DurationMS         int32
	Cost               float64
	CostType           int32
	SpellLevel         int
	// FamilyMask is the client's four mask columns verbatim; they are
	// kept separate rather than folded into one uint64 because the
	// client itself never combines them.
	FamilyMask [4]int64
	Effects    []Effect
}

// EffectiveCooldownMS is the cooldown a player actually experiences: the
// spell's own CooldownMS when it has one, or its CategoryCooldownMS
// otherwise. Vanilla abilities that share a cooldown across their ranks
// (Bloodthirst among them) carry CooldownMS 0 and put the real cooldown
// on the category instead.
func (s Spell) EffectiveCooldownMS() int32 {
	if s.CooldownMS != 0 {
		return s.CooldownMS
	}
	return s.CategoryCooldownMS
}

// rawClass is the JSON envelope of one generated class file.
type rawClass struct {
	Build     string               `json:"build"`
	ClassSlug string               `json:"class_slug"`
	Family    int32                `json:"family"`
	Spells    map[string]spellBody `json:"spells"`
}

// Class is one generated per-class file, spells resolved to a slice and
// sorted by name then rank.
type Class struct {
	Slug   string
	Build  string
	Family int32
	Spells []Spell

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

// Load reads a generated class file and resolves every effect's
// spell-power coefficient.
func Load(path string) (Class, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Class{}, fmt.Errorf("spellconst: %w", err)
	}
	var raw rawClass
	if err := json.Unmarshal(b, &raw); err != nil {
		return Class{}, fmt.Errorf("spellconst: parsing %s: %w", path, err)
	}
	if raw.ClassSlug == "" {
		return Class{}, fmt.Errorf("spellconst: %s has no class_slug", path)
	}
	if raw.Build == "" {
		return Class{}, fmt.Errorf("spellconst: %s has no build; a constants file must record which client it came from", path)
	}

	c := Class{
		Slug:   raw.ClassSlug,
		Build:  raw.Build,
		Family: raw.Family,
		hybrid: hybridClasses[raw.ClassSlug],
	}

	c.Spells = make([]Spell, 0, len(raw.Spells))
	for key, body := range raw.Spells {
		id, err := strconv.ParseInt(key, 10, 32)
		if err != nil {
			return Class{}, fmt.Errorf("spellconst: %s: spell key %q is not an integer id: %w", path, key, err)
		}
		s := Spell{
			ID:                 int32(id),
			Name:               strings.TrimSpace(body.Name),
			Rank:               body.Rank,
			SchoolMask:         body.SchoolMask,
			CastTimeMS:         body.CastTimeMS,
			GCDMS:              body.GCDMS,
			CooldownMS:         body.CooldownMS,
			CategoryCooldownMS: body.CategoryCooldownMS,
			DurationMS:         body.DurationMS,
			Cost:               body.Cost,
			CostType:           body.CostType,
			SpellLevel:         body.SpellLevel,
			FamilyMask:         body.FamilyMask,
			Effects:            append([]Effect(nil), body.Effects...),
		}
		for i := range s.Effects {
			e := &s.Effects[i]
			if e.SPCoefficient != 0 {
				e.ResolvedSPCoefficient = e.SPCoefficient
				e.CoefficientSource = "table"
				continue
			}
			e.ResolvedSPCoefficient, e.CoefficientSource = CoefficientFor(s.CastTimeMS, s.DurationMS, c.hybrid)
		}
		c.Spells = append(c.Spells, s)
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
