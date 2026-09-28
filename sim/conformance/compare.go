package conformance

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// Row is one spell rank, at one level, of one spec, compared against the
// client's constants for its class.
type Row struct {
	ClassSlug string
	Spec      string
	Level     int32

	SpellName string
	Rank      int
	SpellID   int32

	ClientCostAmt  float64
	ClientCostType string
	EngineCostAmt  float64
	EngineCostType string

	ClientCooldownMS int32
	EngineCooldownMS int32

	ClientCastTimeMS int32
	EngineCastTimeMS int32

	ClientGCDMS int32
	EngineGCDMS int32

	ClientRequiredLevel int
	EngineRequiredLevel int

	// ClientDurationMS is the client's duration_ms verbatim: -1 means "until
	// removed" (a stance, a permanent aura), 0 means the client states no
	// duration, and a positive value is milliseconds.
	ClientDurationMS int32
	// EngineDurationMS is 0 when the spell has neither a RelatedSelfBuff nor
	// a Dot the engine can read without running the sim (see rowFor).
	EngineDurationMS int32
	// HasDuration is false when neither side names a duration for this
	// spell, so the duration columns and Verdict ignore each other.
	HasDuration bool

	// Verdict is "match", "mismatch", or "client-scripted" (every field
	// besides duration matches, and the client's own duration is absent
	// while the engine keeps Classic's known value).
	Verdict string
	// Diff names every field that differed, empty for "match".
	Diff string
}

// costTypeName maps both sides' cost-type encodings to one comparable
// label. The client's cost_type column (0 mana, 1 rage, 3 energy, per the
// Forever spellconst pipeline) and the engine's core.CostType enum (Mana=1,
// Energy=2, Rage=3, Focus=4) do not share numbering, so the mapping goes
// through this shared vocabulary rather than comparing raw ints.
func clientCostTypeName(t int32) string {
	switch t {
	case 0:
		return "mana"
	case 1:
		return "rage"
	case 2:
		return "focus"
	case 3:
		return "energy"
	default:
		return fmt.Sprintf("client:%d", t)
	}
}

// normalizeClientCost undoes the client table's own storage convention for
// rage: rage is a fixed-point resource internally (a bar of 0-1000
// representing 0-100.0 rage), so the client's cost column for a rage spell
// is ten times the rage a player actually spends (Execute's 150 is 15
// rage, Bloodthirst's 300 is 30) — confirmed against every rage ability's
// well-known Classic cost. Mana and energy costs are already in the units
// a player sees, so only cost_type 1 is scaled.
func normalizeClientCost(cost float64, costType int32) float64 {
	if costType == 1 {
		return cost / 10
	}
	return cost
}

func engineCostTypeName(t core.CostType) string {
	switch t {
	case core.CostTypeMana:
		return "mana"
	case core.CostTypeEnergy:
		return "energy"
	case core.CostTypeRage:
		return "rage"
	case core.CostTypeFocus:
		return "focus"
	default:
		return "none"
	}
}

// rowFor compares one engine spell against its client entry. ok is false
// when the spell has no real ActionID.SpellID (e.g. OtherID actions like
// auto attack) or the client's class table has no such id (a shared or
// racial ability, an id this class file does not carry, or an id belonging
// to another class's family) — the golden only carries spells the client
// actually names for this class, per the lane brief.
func rowFor(clientClass spellconst.Class, spec Preset, level int32, spell *core.Spell) (Row, bool) {
	if spell.ActionID.SpellID == 0 {
		return Row{}, false
	}
	clientSpell, ok := clientClass.ByID(spell.ActionID.SpellID)
	if !ok {
		return Row{}, false
	}

	row := Row{
		ClassSlug: spec.ClientClassSlug,
		Spec:      spec.Label,
		Level:     level,
		SpellName: clientSpell.Name,
		Rank:      spell.Rank,
		SpellID:   spell.ActionID.SpellID,

		ClientCostAmt:  normalizeClientCost(clientSpell.Cost, clientSpell.CostType),
		ClientCostType: clientCostTypeName(clientSpell.CostType),

		ClientCooldownMS: clientSpell.EffectiveCooldownMS(),
		ClientCastTimeMS: clientSpell.CastTimeMS,
		ClientGCDMS:      clientSpell.GCDMS,

		ClientRequiredLevel: clientSpell.SpellLevel,
		EngineRequiredLevel: spell.RequiredLevel,

		ClientDurationMS: clientSpell.DurationMS,

		EngineCooldownMS: int32(spell.CD.Duration / time.Millisecond),
		EngineCastTimeMS: int32(spell.DefaultCast.CastTime / time.Millisecond),
		EngineGCDMS:      int32(spell.DefaultCast.GCD / time.Millisecond),
	}

	if spell.Cost != nil {
		row.EngineCostAmt = spell.Cost.BaseCost
		row.EngineCostType = engineCostTypeName(spell.Cost.CostType())
	} else {
		row.EngineCostType = "none"
	}

	row.EngineDurationMS, row.HasDuration = engineDuration(spell)
	if row.ClientDurationMS > 0 {
		row.HasDuration = true
	}

	row.Verdict, row.Diff = verdictFor(row)
	return row, true
}

// engineDuration reads a spell's duration without running a sim: the aura
// it applies to its own caster (RelatedSelfBuff), or failing that the dot
// it applies to the current target (Dot(target)), both of which are
// populated at spell-registration time (sim/core/dot.go's createDots),
// before any iteration runs. A spell with neither is not a duration
// ability at all (an instant nuke, a cooldown-only ability), so
// HasDuration is false rather than 0 meaning "0ms".
func engineDuration(spell *core.Spell) (ms int32, has bool) {
	if spell.RelatedSelfBuff != nil {
		return int32(spell.RelatedSelfBuff.Duration / time.Millisecond), true
	}
	if spell.Dots() != nil && spell.Unit != nil && spell.Unit.CurrentTarget != nil {
		if dot := spell.Dot(spell.Unit.CurrentTarget); dot != nil && dot.Aura != nil {
			return int32(dot.Aura.Duration / time.Millisecond), true
		}
	}
	return 0, false
}

// verdictFor compares every column rowFor filled in and decides match,
// mismatch, or client-scripted. Cost is a match when both sides charge
// nothing, regardless of type (a free spell's client cost_type is often a
// leftover 0/mana even though nothing is actually spent).
func verdictFor(row Row) (verdict string, diff string) {
	var diffs []string

	costMatches := almostEqual(row.ClientCostAmt, row.EngineCostAmt)
	if !costMatches {
		diffs = append(diffs, fmt.Sprintf("cost %.2f->%.2f", row.ClientCostAmt, row.EngineCostAmt))
	}
	costTypeMatters := row.ClientCostAmt != 0 || row.EngineCostAmt != 0
	costTypeMatches := !costTypeMatters || row.ClientCostType == row.EngineCostType
	if costTypeMatters && !costTypeMatches {
		diffs = append(diffs, fmt.Sprintf("cost_type %s->%s", row.ClientCostType, row.EngineCostType))
	}

	cooldownMatches := row.ClientCooldownMS == row.EngineCooldownMS
	if !cooldownMatches {
		diffs = append(diffs, fmt.Sprintf("cooldown_ms %d->%d", row.ClientCooldownMS, row.EngineCooldownMS))
	}

	// A negative client cast_time_ms (observed as exactly -1000000 on
	// ranged Hunter abilities: Arcane Shot, Serpent Sting) is the client's
	// sentinel for "instant, but gated by ranged weapon speed", not a
	// literal cast time in the negative millions of milliseconds; treated
	// as not comparable rather than as a match or a mismatch.
	castMatches := row.ClientCastTimeMS < 0 || row.ClientCastTimeMS == row.EngineCastTimeMS
	if !castMatches {
		diffs = append(diffs, fmt.Sprintf("cast_time_ms %d->%d", row.ClientCastTimeMS, row.EngineCastTimeMS))
	}

	gcdMatches := row.ClientGCDMS == row.EngineGCDMS
	if !gcdMatches {
		diffs = append(diffs, fmt.Sprintf("gcd_ms %d->%d", row.ClientGCDMS, row.EngineGCDMS))
	}

	levelMatches := row.ClientRequiredLevel == row.EngineRequiredLevel
	if !levelMatches {
		diffs = append(diffs, fmt.Sprintf("required_level %d->%d", row.ClientRequiredLevel, row.EngineRequiredLevel))
	}

	durationMatches := true
	clientStatesNoDuration := false
	if row.HasDuration {
		if row.ClientDurationMS <= 0 {
			// The client states no duration (0) or a permanent one (-1
			// stances aside, which this program does not try to score).
			// The engine keeping a known value here is the documented
			// "server-side script" case, not a defect.
			clientStatesNoDuration = row.ClientDurationMS == 0
			durationMatches = row.ClientDurationMS == -1 && row.EngineDurationMS == 0
			if !durationMatches && !clientStatesNoDuration {
				diffs = append(diffs, fmt.Sprintf("duration_ms %d->%d", row.ClientDurationMS, row.EngineDurationMS))
			}
		} else if row.ClientDurationMS != row.EngineDurationMS {
			durationMatches = false
			diffs = append(diffs, fmt.Sprintf("duration_ms %d->%d", row.ClientDurationMS, row.EngineDurationMS))
		}
	}

	otherFieldsMatch := costMatches && costTypeMatches && cooldownMatches && castMatches && gcdMatches && levelMatches

	switch {
	case otherFieldsMatch && (durationMatches || (row.HasDuration && clientStatesNoDuration)):
		if row.HasDuration && clientStatesNoDuration && row.EngineDurationMS != 0 {
			return "client-scripted", "duration_ms: client states none (0), engine keeps " + fmt.Sprintf("%dms", row.EngineDurationMS)
		}
		return "match", ""
	default:
		return "mismatch", joinDiffs(diffs)
	}
}

func joinDiffs(diffs []string) string {
	out := ""
	for i, d := range diffs {
		if i > 0 {
			out += "; "
		}
		out += d
	}
	return out
}

func almostEqual(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.01
}
