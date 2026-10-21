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
	// ClientCostPct is the client's percent-of-base-mana price when it
	// costs the spell that way (SpellPower.PowerCostPct, carried by
	// spellconst as CostPct); ClientCostAmt is then that percentage of
	// the preset character's base mana, the same number the engine's
	// ManaCostOptions.BaseCost resolves to. 0 for a flat-cost spell.
	ClientCostPct  float64
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
	// EngineDurationMS is 0 when engineDuration found no aura or Dot
	// anywhere it looked (see its doc comment for the full search order).
	EngineDurationMS int32
	// EngineDurationFound is true when engineDuration located a real aura
	// or Dot for this spell, on any unit it checked - the caster, the
	// caster's current target, or a pet the caster's Character owns -
	// false when none of those has anything at all. It exists only to
	// tell a genuine "both sides have a number but they disagree"
	// mismatch apart from "this report found no engine-side duration to
	// compare in the first place" in the rendered Diff text; it is not
	// itself rendered as a column.
	EngineDurationFound bool
	// HasDuration is false when neither side names a duration for this
	// spell, so the duration columns and Verdict ignore each other.
	HasDuration bool
	// DurationReading is non-empty when the duration column is read through
	// one of the report-side readings in duration_reading.go instead of
	// being compared as numbers; it holds the reading's explanation.
	DurationReading string

	// Damage is the base-damage comparison (damage.go). It is reported in
	// its own columns and does not move Verdict.
	Damage DamageComparison

	// Verdict is "match", "mismatch", or "client-scripted" (every field
	// besides duration matches, and the client's own duration is absent
	// while the engine keeps Classic's known value).
	Verdict string
	// Diff names every field that differed, empty for "match".
	Diff string
}

// Percent-of-base-mana costs: the client prices some spells through
// SpellPower.PowerCostPct (Arcane Blast 15, Judgement 6, Shadowform 40,
// the warlock summons 80/100, Multi-Shot 13.9, the druid forms) and
// leaves the flat ManaCost column 0 for them. Until 2026-10-07 spellconst
// carried only the flat column, so such a spell registered free by the
// engine scored a clean "0 -> 0 match" here (Arcane Blast, Judgement,
// Shadowform, Divine Favor, Bane of Havoc and Strider Kick all did). The
// data pipeline now emits cost_pct and spellconst.Spell carries it as
// CostPct; rowFor resolves it against the preset character's base mana
// (ClientCostPct, ClientCostAmt) so the cost column compares the same
// amount newManaCost derives from ManaCostOptions.BaseCost, and the
// golden prints the percentage beside it.
//
// Shaman's Stormstrike (17364) is NOT one of these: its raw
// SpellPower.csv row has PowerCostPct 0 and a real flat ManaCost of
// 125, matching spellconst's own cost column already. The row still
// mismatches (client 125 vs. engine 319.20) because
// sim/shaman/stormstrike.go's ManaCost.BaseCost: .21 treats it as a
// 21%-of-base-mana spell anyway - a stale vanilla-era literal, not a
// visibility gap, and the shaman lane's to fix.
//
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
//
// A spell flagged SpellFlagPassiveSpell ("applied/cast as a result of
// another spell", sim/core/flags.go) is also skipped: it is an internal
// helper the ability's own ApplyEffects drives directly (Death Coil's
// self-heal, sim/warlock/death_coil.go, registers one with the SAME
// SpellID as the cast itself, just a different ActionID Tag), not a
// player action the client's per-rank table describes cost/cooldown/cast
// time for. Without this, that helper produced its own all-zero row
// against the real cast's client entry - Death Coil looked
// "unregistered" in the golden even where the real cast matched.
func rowFor(clientClass spellconst.Class, spec Preset, level int32, character *core.Character, spell *core.Spell) (Row, bool) {
	if spell.ActionID.SpellID == 0 {
		return Row{}, false
	}
	if spell.Flags.Matches(core.SpellFlagPassiveSpell) {
		return Row{}, false
	}
	clientSpell, ok := clientClass.ByID(spell.ActionID.SpellID)
	if !ok {
		return Row{}, false
	}

	// JOB 3: the client's per-class table carries a generic "Attack"
	// entry - the basic melee-swing button, not a player-cast spell -
	// reused verbatim across many contexts. It sometimes even shares a
	// SpellID with a real engine-registered ability (shaman's Searing
	// Totem reuses "Attack"'s ids for its own instant totem-attack
	// sub-spell), which produced a spurious mismatch comparing the
	// client's melee-swing timer against that ability's real cast time.
	// It was never meant to be compared against any one engine spell, so
	// this report excludes it rather than scoring it.
	if clientSpell.Name == "Attack" {
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

		ClientDurationMS: durationSpellFor(clientClass, clientSpell).DurationMS,

		EngineCooldownMS: int32(spell.CD.Duration / time.Millisecond),
		EngineCastTimeMS: int32(spell.DefaultCast.CastTime / time.Millisecond),
		EngineGCDMS:      int32(spell.DefaultCast.GCD / time.Millisecond),
	}

	// A percent-of-base-mana price (Arcane Blast 15%, Judgement 6%) has a
	// flat client cost of 0, which until 2026-10-07 read as a clean
	// "0->0 match" against an engine that charged nothing - the exact
	// defect this column now catches. The client percentage is resolved
	// against the preset character's own base mana, which is what
	// newManaCost (sim/core/mana.go) multiplies BaseCost by.
	if clientSpell.CostPct > 0 && clientSpell.CostType == 0 && spell.Unit != nil {
		row.ClientCostPct = clientSpell.CostPct
		row.ClientCostAmt = clientSpell.CostPct / 100 * spell.Unit.BaseMana
	}

	if spell.Cost != nil {
		row.EngineCostAmt = spell.Cost.BaseCost
		row.EngineCostType = engineCostTypeName(spell.Cost.CostType())
	} else {
		row.EngineCostType = "none"
	}

	row.EngineDurationMS, row.EngineDurationFound = engineDuration(spell, character, siblingSpellIDs(clientClass, clientSpell.Name))
	row.HasDuration = row.EngineDurationFound
	if row.ClientDurationMS > 0 {
		row.HasDuration = true
	}

	row.DurationReading = durationReading(row, durationSpellFor(clientClass, clientSpell), spell, siblingSpellIDs(clientClass, clientSpell.Name))

	row.Damage = compareDamage(clientSpell, int(level), spell)

	row.Verdict, row.Diff = verdictFor(row)
	return row, true
}

// siblingSpellIDs returns every SpellID spellconst's client table lists
// under name - every rank of the same ability, not just the one rank
// being compared - so engineDuration can match an aura that is pinned to
// one fixed rank's SpellID regardless of which rank actually cast it
// (see engineDuration's doc comment, step 3-5, for why that happens).
func siblingSpellIDs(clientClass spellconst.Class, name string) map[int32]bool {
	ranks := clientClass.Ranks(name)
	ids := make(map[int32]bool, len(ranks))
	for _, r := range ranks {
		ids[r.ID] = true
	}
	return ids
}

// engineDuration locates, without running a sim, the aura or Dot a
// spell's effect actually lives on. found is false only when none of
// the following has anything for this spell at all:
//
//  1. RelatedSelfBuff - a buff the spell links to itself at
//     registration time (sim/core/dot.go's createDots, before any
//     iteration runs). Checked first, alone, exactly as before this
//     report looked any further: most registered durations live here.
//  2. Dot(currentTarget) - a damage-over-time aura on the sim's one
//     default target, populated at that same registration time.
//  3. Every aura already registered on the spell's own caster
//     (Unit.GetAuras()) - this is where a totem's lifetime lives in
//     this codebase: totems are not separate units here, they are
//     auras the totem's own ApplyEffects activates directly on the
//     shaman's Unit (core.StrengthOfEarthTotemAura and its siblings in
//     sim/core/buffs.go), and a self buff registered without
//     RelatedSelfBuff (Rapid Fire, Berserk, Slice and Dice, Shield
//     Wall) also lives here.
//  4. Every aura already registered on the spell's current target - a
//     target debuff that is not itself a Dot (Earth Shock's interrupt
//     silence, Frost Shock's slow, Expose Armor, Faerie Fire).
//  5. Every aura registered on a Pet the caster's Character owns - the
//     one other unit type this report can reach, for any class whose
//     minion or totem is modeled as a real Pet rather than a
//     caster-side aura.
//
// Steps 3-5 match an aura by ActionID.SpellID against siblingIDs
// (siblingSpellIDs), not against spell.ActionID.SpellID alone, because
// several of the engine's own shared-aura helpers register one aura
// pinned to a single fixed rank's SpellID no matter which rank actually
// triggered it.
//
// A spell with no duration ability at all (an instant nuke, a
// cooldown-only button) is indistinguishable from one whose armed
// lifetime this report cannot see (a trap before it triggers) by this
// function alone - rowFor's HasDuration/ClientDurationMS combination,
// not found, is what tells those apart for the golden.
func engineDuration(spell *core.Spell, character *core.Character, siblingIDs map[int32]bool) (ms int32, found bool) {
	if spell.RelatedSelfBuff != nil {
		return auraDurationMS(spell.RelatedSelfBuff), true
	}
	// Any target's dot carries the same duration; the caster's current
	// target is not always an enemy (a healer's is a friend).
	for _, dot := range spell.Dots() {
		if dot != nil && dot.Aura != nil {
			return auraDurationMS(dot.Aura), true
		}
	}

	if spell.Unit == nil {
		return 0, false
	}

	if aura := matchingAura(spell.Unit.GetAuras(), siblingIDs); aura != nil {
		return auraDurationMS(aura), true
	}
	for _, target := range enemyTargets(spell.Unit) {
		if aura := matchingAura(target.GetAuras(), siblingIDs); aura != nil {
			return auraDurationMS(aura), true
		}
	}
	if character != nil {
		for _, pet := range character.Pets {
			if aura := matchingAura(pet.GetAuras(), siblingIDs); aura != nil {
				return auraDurationMS(aura), true
			}
		}
	}
	return 0, false
}

// enemyTargets is the caster's current target, when it has one, followed
// by every target of the encounter: a healing spec has no current target
// (its debuff spells, such as Holy's Judgement of the Crusader, still
// register their auras on the encounter's targets).
func enemyTargets(caster *core.Unit) []*core.Unit {
	var targets []*core.Unit
	if caster.CurrentTarget != nil {
		targets = append(targets, caster.CurrentTarget)
	}
	if caster.Env != nil {
		for _, target := range caster.Env.Encounter.TargetUnits {
			if target != caster.CurrentTarget {
				targets = append(targets, target)
			}
		}
	}
	return targets
}

// auraDurationMS converts an aura's Duration to the client's own
// millisecond convention, including its sentinel: core.NeverExpires (a
// proc-style aura with no natural timeout, such as Inner Focus or
// Shadowform, removed by code rather than by expiring) is reported as
// -1, the client's own "until removed" value, rather than truncating
// MaxInt64 nanoseconds into an int32 and returning millisecond garbage.
func auraDurationMS(aura *core.Aura) int32 {
	if aura.Duration == core.NeverExpires {
		return -1
	}
	return int32(aura.Duration / time.Millisecond)
}

// matchingAura returns the first aura in auras whose ActionID.SpellID is
// a member of ids, or nil. Tag is deliberately ignored: the search
// cares only which spell an aura belongs to, not which proc or stack
// instance of it this particular Aura value is.
func matchingAura(auras []*core.Aura, ids map[int32]bool) *core.Aura {
	for _, aura := range auras {
		if aura.ActionID.SpellID != 0 && ids[aura.ActionID.SpellID] {
			return aura
		}
	}
	return nil
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
	if row.DurationReading != "" {
		// Read through a report-side reading (duration_reading.go): the
		// numbers are not comparable, so they are not scored.
	} else if row.HasDuration {
		if row.ClientDurationMS <= 0 {
			// The client states no duration (0) or a permanent one (-1
			// stances aside, which this program does not try to score).
			// The engine keeping a known value here is the documented
			// "server-side script" case, not a defect. A client -1 also
			// matches an engine 0: before Job 1 widened engineDuration's
			// search, "not found" (0) was the only way a permanent buff
			// ever read here at all, and that convention stays valid for
			// whatever still reads 0 today (nothing found, anywhere).
			clientStatesNoDuration = row.ClientDurationMS == 0
			durationMatches = row.ClientDurationMS == -1 && (row.EngineDurationMS == 0 || row.EngineDurationMS == -1)
			if !durationMatches && !clientStatesNoDuration {
				diffs = append(diffs, fmt.Sprintf("duration_ms %d->%d%s", row.ClientDurationMS, row.EngineDurationMS, noAuraNote(row)))
			}
		} else if row.ClientDurationMS != row.EngineDurationMS {
			durationMatches = false
			diffs = append(diffs, fmt.Sprintf("duration_ms %d->%d%s", row.ClientDurationMS, row.EngineDurationMS, noAuraNote(row)))
		}
	}

	otherFieldsMatch := costMatches && costTypeMatches && cooldownMatches && castMatches && gcdMatches && levelMatches

	switch {
	case otherFieldsMatch && row.DurationReading != "":
		return VerdictUnmodeledDuration, "duration_ms: " + row.DurationReading
	case otherFieldsMatch && (durationMatches || (row.HasDuration && clientStatesNoDuration)):
		if row.HasDuration && clientStatesNoDuration && row.EngineDurationMS != 0 {
			return "client-scripted", "duration_ms: client states none (0), engine keeps " + fmt.Sprintf("%dms", row.EngineDurationMS)
		}
		return "match", ""
	default:
		return "mismatch", joinDiffs(diffs)
	}
}

// noAuraNote annotates a duration mismatch that engineDuration could not
// back with any real aura or Dot - " (no aura registered)" - versus a
// mismatch where an aura was found but the two sides' numbers simply
// disagree, which gets no note. This is the "new column or note" the
// lane brief calls for: a trap's armed lifetime (Explosive Trap,
// Freezing Trap - see SUMMARY.md's Hunter section) is the known case
// still reading this way after engineDuration's wider search; anything
// a future registration adds a real aura for stops reading it
// automatically, with no further change here.
func noAuraNote(row Row) string {
	if row.EngineDurationFound {
		return ""
	}
	return " (no aura registered)"
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
