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
	"bytes"
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

// rawClass is the JSON envelope of one generated class file. Spells is
// kept as raw JSON per entry, not decoded directly into spellBody here,
// because each entry needs two independent checks before it is trusted:
// a required-field presence check (a struct field alone cannot tell "0"
// from "absent") and an unknown-field rejection (via a second decode
// with DisallowUnknownFields), both against the same bytes.
type rawClass struct {
	Build     string                     `json:"build"`
	ClassSlug string                     `json:"class_slug"`
	Family    int32                      `json:"family"`
	Spells    map[string]json.RawMessage `json:"spells"`
}

// requiredTopLevelFields, requiredSpellFields and requiredEffectFields
// are every field the amended `simconst` contract
// (docs/superpowers/specs/2026-09-14-simulator-interfaces.md) marks as
// part of the shape, at each of its three levels. A field missing from
// the JSON is a pipeline defect worth failing loudly on — the contract
// changed shape on this package once already (an array became an
// object) without the loader noticing, which is why this package exists
// as a follow-up at all.
var requiredTopLevelFields = []string{"build", "class_slug", "family", "spells"}

var requiredSpellFields = []string{
	"name", "rank", "school_mask", "cast_time_ms", "gcd_ms", "cooldown_ms",
	"category_cooldown_ms", "duration_ms", "cost", "cost_type", "spell_level",
	"family_mask", "effects",
}

var requiredEffectFields = []string{
	"index", "effect", "aura", "amount", "sp_coefficient", "ap_coefficient",
	"period_ms", "misc_value", "trigger_spell",
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

// hybridClasses are the classes the vanilla convention halves. Every
// other class (including priest — shadow priests use the full
// convention, not the healer half) is considered non-hybrid by the zero
// value a missing map key already returns, so only the three true
// entries are listed.
var hybridClasses = map[string]bool{
	"paladin": true,
	"shaman":  true,
	"druid":   true,
}

// Load reads a generated class file, rejecting an unrecognized field or
// a field the contract requires but the file omits, and resolves every
// effect's spell-power coefficient.
func Load(path string) (Class, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Class{}, fmt.Errorf("spellconst: %w", err)
	}

	if err := requireFields(b, requiredTopLevelFields); err != nil {
		return Class{}, fmt.Errorf("spellconst: %s: %w", path, err)
	}

	var raw rawClass
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
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

	// Iterate the spell ids in a fixed, sorted order rather than the
	// map's own (randomized per run) order. Nothing downstream currently
	// depends on load order for correctness — the generator resolves any
	// same-(name,rank) collision explicitly — but a loader whose output
	// order is not reproducible is a bug waiting for the next thing that
	// assumes it is, and there is no cost to fixing it here once.
	ids := make([]string, 0, len(raw.Spells))
	for key := range raw.Spells {
		ids = append(ids, key)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, _ := strconv.ParseInt(ids[i], 10, 64)
		b, _ := strconv.ParseInt(ids[j], 10, 64)
		return a < b
	})

	c.Spells = make([]Spell, 0, len(raw.Spells))
	for _, key := range ids {
		spellRaw := raw.Spells[key]
		id, err := strconv.ParseInt(key, 10, 32)
		if err != nil {
			return Class{}, fmt.Errorf("spellconst: %s: spell key %q is not an integer id: %w", path, key, err)
		}

		if err := requireFields(spellRaw, requiredSpellFields); err != nil {
			return Class{}, fmt.Errorf("spellconst: %s: spell %d: %w", path, id, err)
		}
		effectsRaw, err := effectMessages(spellRaw)
		if err != nil {
			return Class{}, fmt.Errorf("spellconst: %s: spell %d: %w", path, id, err)
		}
		for i, er := range effectsRaw {
			if err := requireFields(er, requiredEffectFields); err != nil {
				return Class{}, fmt.Errorf("spellconst: %s: spell %d effect %d: %w", path, id, i, err)
			}
		}

		var body spellBody
		bodyDec := json.NewDecoder(bytes.NewReader(spellRaw))
		bodyDec.DisallowUnknownFields()
		if err := bodyDec.Decode(&body); err != nil {
			return Class{}, fmt.Errorf("spellconst: %s: spell %d: %w", path, id, err)
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

// requireFields checks that every name in want is a key of the JSON
// object in raw, returning an error naming the first one missing. It
// does not care about the value, only that the key was present — a
// legitimately zero cast time and an omitted cast_time_ms are otherwise
// indistinguishable once decoded into a struct.
func requireFields(raw json.RawMessage, want []string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("not a JSON object: %w", err)
	}
	for _, name := range want {
		if _, ok := fields[name]; !ok {
			return fmt.Errorf("missing required field %q", name)
		}
	}
	return nil
}

// effectMessages splits a spell's raw JSON into its effects array's raw
// entries, for requireFields to check each one independently. A spell
// missing its effects key entirely already failed requireFields against
// requiredSpellFields before this is called.
func effectMessages(raw json.RawMessage) ([]json.RawMessage, error) {
	var fields struct {
		Effects []json.RawMessage `json:"effects"`
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("parsing effects: %w", err)
	}
	return fields.Effects, nil
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
