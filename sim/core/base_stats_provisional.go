package core

import "github.com/wowsims/classic/sim/core/proto"

// base_stats_auto_gen.go is generated from combatratings.txt, one of the
// three GameTables files the data lane has confirmed exist on the Classic
// lineage (the other two, basemp.txt and hppersta.txt, carry nothing this
// package reads). Its four defensive rating constants and BaseStatsBuild
// are therefore Forever's own measured values, not Era's, and are not
// listed below.
//
// What is NOT covered by any of the three GameTables files is the
// per-race, per-class base-stat table: ClassBaseStats and RaceOffsets in
// base_stats.go are still hand-typed Era numbers (research/08-stats.md
// §12.4 item 4), because no basestats/ mining directory exists for build
// 1.60.1.69893 - the three GameTables are the entire yield of that build's
// data pull. Forever ships ten races; the Era table only ever had eight.
// This file names that gap out loud rather than letting a Skyborne (or
// any) sim present those numbers as measured.
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
// The rating constants generated in base_stats_auto_gen.go are NOT here:
// research/08-stats.md §2 and §12.4 settle Crit/Hit/Haste/Expertise at a
// flat 1:1 (confirmed, not measured, by design), and Defense/Dodge/Parry/
// Block now come straight out of combatratings.txt for build 1.60.1.69893
// (confirmed, measured). Only the per-race/class base-stat table -
// entirely untouched Era data - and the two Skyborne rows cloned from it
// remain unconfirmed.
var provisionalConstantNames = []string{
	"ClassBaseStats/RaceOffsets",        // unconfirmed: no basestats table exists for build 1.60.1.69893; values are still Era's
	"BaseStats[RaceHighOrderSkyborne]",  // unconfirmed: Skyborne base stats are not in any mined table; cloned from Human
	"BaseStats[RaceWindshaperSkyborne]", // unconfirmed: Skyborne base stats are not in any mined table; cloned from Orc
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
func init() {
	// RaceOffsets and BaseStats are declared in base_stats.go; this file
	// only adds entries, it does not redefine either map.
	RaceOffsets[proto.Race_RaceHighOrderSkyborne] = RaceOffsets[proto.Race_RaceHuman] // unconfirmed: cloned from Human
	RaceOffsets[proto.Race_RaceWindshaperSkyborne] = RaceOffsets[proto.Race_RaceOrc]  // unconfirmed: cloned from Orc

	// BaseStats (keyed by race, class and level) is otherwise unpopulated
	// until a mined per-level table exists. Fill it at level 60 - the
	// level ClassBaseStats and RaceOffsets already represent - for every
	// race (including the two Skyborne clones just above) and class, so
	// a Skyborne lookup is a genuine clone of a computed row rather than
	// two absent map entries comparing equal by accident.
	for race := range RaceOffsets {
		for class := range ClassBaseStats {
			BaseStats[BaseStatsKey{Race: race, Class: class, Level: 60}] = getBaseStatsCombo(race, class)
		}
	}
}
