package core

import (
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

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
	"BaseStats[RaceHighOrderSkyborne]",                                                  // unconfirmed: the beta character sheet; the launch build gives each Skyborne variant its own offsets, not yet read
	"BaseStats[RaceWindshaperSkyborne]",                                                 // unconfirmed: the beta character sheet; the launch build gives each Skyborne variant its own offsets, not yet read
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

// Skyborne base stats are in no client table: ChrRaces 1.60.1.70755 has
// no stat columns, RaceStat carries one zero column for every race, and
// no ChrClassRaceSex table ships (checked on wago.tools for 70755 and
// 70790, 2026-11-04). The only measured Forever value is the beta
// character sheet: a naked level-1 Skyborne reads Human -1 Strength,
// +1 Agility, -1 Stamina, +1 Intellect and +0 Spirit, for both variants
// (ClassicWoWCommunity/forever-bugs #414, 2026-10-20, naked level-1
// sheets; Warcraft Tavern's Skyborne guide prints the same sheet for all
// six classes). Blizzard's reply on #414 says the launch build gives each
// variant its own, larger offsets "following the same rules as the
// original stat offsets", which nobody has published: until a launch
// sheet is read, both variants carry the beta row and stay in
// ProvisionalConstants. Before round 20 High Order cloned Human and
// Windshaper cloned Orc, which no source ever supported.
//
// This only fills RaceOffsets, the map getBaseStatsCombo actually reads
// (character.go:153). It does not also populate the BaseStats[race,
// class, level] map declared in base_stats.go: that map is unread by any
// runtime path in this fork. TestSkyborneBaseStatsFollowTheBetaCharacterSheet
// pins the row and refuses a Human or Orc clone.
var skyborneBetaSheetOffsets = stats.Stats{
	stats.Strength:  -1,
	stats.Agility:   1,
	stats.Stamina:   -1,
	stats.Intellect: 1,
	stats.Spirit:    0,
}

func init() {
	// RaceOffsets is declared in base_stats.go; this file only adds
	// entries to it, it does not redefine the map.
	RaceOffsets[proto.Race_RaceHighOrderSkyborne] = skyborneBetaSheetOffsets  // unconfirmed: beta sheet; launch offsets per variant not yet read
	RaceOffsets[proto.Race_RaceWindshaperSkyborne] = skyborneBetaSheetOffsets // unconfirmed: beta sheet; launch offsets per variant not yet read
}
