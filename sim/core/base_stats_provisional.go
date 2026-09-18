package core

import "github.com/wowsims/classic/sim/core/proto"

// base_stats_auto_gen.go is generated from the three GameTables files the
// data lane has confirmed exist on the Classic lineage for build
// 1.60.1.69893: combatratings.txt, basemp.txt and hppersta.txt. Forever
// has no combat-rating system (research/08-stats.md §2: flat percentages,
// settled), so combatratings.txt is read and recorded but not applied -
// every rating constant is a confirmed flat 1. basemp.txt and hppersta.txt
// do carry live data: their build-1.60.1.69893 values confirm (but do not
// redefine) ClassBaseStats' Mana fields and character.go's Stamina->Health
// dependency - see base_stats_auto_gen.go's header and
// TestBaseManaAndHealthPerStaminaConfirmedAgainstTheTable.
//
// What is NOT covered by any of the three GameTables files is the rest of
// the per-race, per-class base-stat table: ClassBaseStats' Str/Agi/Sta/
// Int/Spirit/Health/AttackPower fields and RaceOffsets in base_stats.go
// are still hand-typed Era numbers (research/08-stats.md §12.4 item 4),
// because no basestats/ mining directory exists for build 1.60.1.69893 -
// the three GameTables are the entire yield of that build's data pull.
// Forever ships ten races; the Era table only ever had eight. This file
// names that gap out loud rather than letting a Skyborne (or any) sim
// present those numbers as measured.
//
// Clearing an entry off provisionalConstantNames is a data-arrival event,
// not a code change:
//
//  1. Land a mined basestats/ table for the missing race/class rows.
//  2. Extend tools/base_stats_parser.py to read it and regenerate
//     ClassBaseStats/RaceOffsets (or their replacement), or hand-fill the
//     two Skyborne rows from real data instead of a clone.
//  3. Delete the corresponding line below.
//
// The nightly validation job (api lane) is what proves the result.

// provisionalConstantNames are the constants whose values are not backed
// by any of the three confirmed GameTables files and are therefore
// unconfirmed for Forever.
//
// Not here: the eight rating constants in base_stats_auto_gen.go (all
// confirmed flat 1:1 by research/08-stats.md §2/§12.4 - four of them
// because combatratings.txt is deliberately not applied, not because it
// is missing), and ClassBaseStats' Mana field (confirmed by basemp.txt).
// Only the remaining per-race/class base-stat fields - entirely untouched
// Era data - and the two Skyborne rows cloned from them remain
// unconfirmed.
var provisionalConstantNames = []string{
	"ClassBaseStats (Str/Agi/Sta/Int/Spirit/Health/AttackPower fields) and RaceOffsets", // unconfirmed: no basestats table exists for build 1.60.1.69893; values are still Era's
	"BaseStats[RaceHighOrderSkyborne]",                                                  // unconfirmed: Skyborne base stats are not in any mined table; cloned from Human
	"BaseStats[RaceWindshaperSkyborne]",                                                 // unconfirmed: Skyborne base stats are not in any mined table; cloned from Orc
}

// ProvisionalConstants returns the names of every constant still carrying
// an unconfirmed (non-Forever-measured) value. It is a static list rather
// than a build-prefix check: BaseStatsBuild names the source of the
// rating constants only, and says nothing about whether the per-race/
// class base-stat table has been mined, so the two data sources are
// tracked independently. It shrinks by deleting an entry as each table
// lands (see the package comment above), never by a version comparison
// that could silently mark un-mined data as confirmed.
func ProvisionalConstants() []string {
	out := make([]string, len(provisionalConstantNames))
	copy(out, provisionalConstantNames)
	return out
}

// Skyborne base stats are not in any mined table (see the package comment
// above). RaceHighOrderSkyborne clones the Human row and
// RaceWindshaperSkyborne clones the Orc row, purely because those are the
// two factions' baseline and nothing better is known. Cloning here -
// rather than hand-typing numbers - means the day a real table lands,
// TestSkyborneBaseStatsAreDeclaredClones fails and the diff is obvious.
//
// This only clones RaceOffsets, the map getBaseStatsCombo actually reads
// (character.go:153). It does not also populate the BaseStats[race,
// class, level] map declared in base_stats.go: that map is unread by any
// runtime path in this fork, and standing ruling (Task 9) is that no
// production behaviour - including a package init() - exists purely to
// give a test something to compare. TestSkyborneBaseStatsAreDeclaredClones
// asserts the clone directly against RaceOffsets and getBaseStatsCombo
// instead.
func init() {
	// RaceOffsets is declared in base_stats.go; this file only adds
	// entries to it, it does not redefine the map.
	RaceOffsets[proto.Race_RaceHighOrderSkyborne] = RaceOffsets[proto.Race_RaceHuman] // unconfirmed: cloned from Human
	RaceOffsets[proto.Race_RaceWindshaperSkyborne] = RaceOffsets[proto.Race_RaceOrc]  // unconfirmed: cloned from Orc
}
