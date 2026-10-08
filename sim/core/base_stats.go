package core

import (
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

type BaseStatsKey struct {
	Race  proto.Race
	Class proto.Class
	Level int
}

var BaseStats = map[BaseStatsKey]stats.Stats{}

// To calculate base stats, get a naked toon of desired level of the race/class you want, ideally without any talents to mess up base stats.
//  Basic stats are as-shown (str/agi/stm/int/spirit)

// Base Spell Crit is calculated by
//   1. Take as-shown value (troll shaman have 3.5%)
//   2. Calculate the bonus from int (for troll shaman that would be 104/78.1=1.331% crit)
//   3. Subtract as-shown from int bouns (3.5-1.331=2.169)
//   4. 2.169*22.08 (rating per crit percent) = 47.89 crit rating.

// Base mana can be looked up here: https://wowwiki-archive.fandom.com/wiki/Base_mana

// These are also scattered in various dbc/casc files,
// `octbasempbyclass.txt`, `combatratings.txt`, `chancetospellcritbase.txt`, etc.

var RaceOffsets = map[proto.Race]stats.Stats{
	proto.Race_RaceUnknown: {},
	proto.Race_RaceHuman:   {},
	proto.Race_RaceOrc: {
		stats.Agility:   -3,
		stats.Strength:  3,
		stats.Intellect: -3,
		stats.Spirit:    3,
		stats.Stamina:   2,
	},
	proto.Race_RaceDwarf: {
		stats.Agility:   -4,
		stats.Strength:  2,
		stats.Intellect: -1,
		stats.Spirit:    -1,
		stats.Stamina:   3,
	},
	proto.Race_RaceNightElf: {
		stats.Agility:   5,
		stats.Strength:  -3,
		stats.Intellect: 0,
		stats.Spirit:    0,
		stats.Stamina:   -1,
	},
	proto.Race_RaceUndead: {
		stats.Agility:   -2,
		stats.Strength:  -1,
		stats.Intellect: -2,
		stats.Spirit:    5,
		stats.Stamina:   1,
	},
	proto.Race_RaceTauren: {
		stats.Agility:   -5,
		stats.Strength:  5,
		stats.Intellect: -5,
		stats.Spirit:    2,
		stats.Stamina:   2,
	},
	proto.Race_RaceGnome: {
		stats.Agility:   3,
		stats.Strength:  -5,
		stats.Intellect: 3,
		stats.Spirit:    0,
		stats.Stamina:   -1,
	},
	proto.Race_RaceTroll: {
		stats.Agility:   2,
		stats.Strength:  1,
		stats.Intellect: -4,
		stats.Spirit:    1,
		stats.Stamina:   1,
	},
}

// Forever has one Crit stat where Era had two, so each class's single
// value here is max(SpellCrit, MeleeCrit) over Era's pair - except where
// the melee column cannot be believed, which is explained next.
//
// untrustworthy: upstream's MeleeCrit column is its Dodge column
// duplicated. In master's ClassBaseCrit (sim/core/base_stats.go at
// master:84-130) MeleeCrit equals Dodge for all nine classes - paladin
// 0.7/0.7, priest 3.0/3.0, shaman 1.7/1.7, mage 3.2/3.2, warlock 2.0/2.0,
// druid 0.9/0.9 and 0/0 for warrior, hunter and rogue. Nine exact matches
// is a copy-paste, not nine coincidences, and a plain max() over it hands
// the mage +3.0 base crit where its spell value is 0.2 and the priest
// +2.2. So for the three rows where the melee value both equals Dodge and
// exceeds the spell value - priest, mage, warlock - the spell value is
// taken and the melee value is discarded; each is marked below. For the
// other six the max() is unaffected: either the spell value already wins
// (paladin, hunter, shaman, druid) or both are zero (warrior, rogue).
// Dodge itself is kept as Era's, since the melee-crit duplication says
// nothing about which of the two columns is the real Dodge.
//
// unconfirmed: chancetomeleecrit.txt and chancetospellcrit.txt do not
// exist for build 1.60.1.69893 - they 404 on wago and are not among the
// three GameTables files the data lane has mined (combatratings.txt,
// basemp.txt, hppersta.txt; see base_stats_provisional.go). Every value
// in this table is therefore still Era's. Named once here, on the table,
// rather than repeated on each of the nine class entries below.
var ClassBaseCrit = map[proto.Class]stats.Stats{
	proto.Class_ClassUnknown: {},
	proto.Class_ClassWarrior: {
		// Forever: merged from SpellCrit 0.0000 + MeleeCrit 0.0000, max().
		stats.Crit:  0.0000 * CritRatingPerCritChance,
		stats.Dodge: 0.0000 * DodgeRatingPerDodgeChance,
	},
	proto.Class_ClassPaladin: {
		// Forever: merged from SpellCrit 3.5000 + MeleeCrit 0.7000, max().
		stats.Crit:  3.5000 * CritRatingPerCritChance,
		stats.Dodge: 0.7000 * DodgeRatingPerDodgeChance,
	},
	proto.Class_ClassHunter: {
		// Forever: merged from SpellCrit 3.6000 + MeleeCrit 0.0000, max().
		stats.Crit:  3.6000 * CritRatingPerCritChance,
		stats.Dodge: 0.0000 * DodgeRatingPerDodgeChance,
	},
	proto.Class_ClassRogue: {
		// Forever: merged from SpellCrit 0.0000 + MeleeCrit 0.0000, max().
		stats.Crit:  0.0000 * CritRatingPerCritChance,
		stats.Dodge: 0.0000 * DodgeRatingPerDodgeChance,
	},
	proto.Class_ClassPriest: {
		// Forever: merged from SpellCrit 0.8000 + MeleeCrit 3.0000. The melee
		// value is the Dodge column duplicated (see the note on the table),
		// so it is discarded and the spell value taken rather than max()'d.
		stats.Crit:  0.8000 * CritRatingPerCritChance,
		stats.Dodge: 3.0000 * DodgeRatingPerDodgeChance,
	},
	proto.Class_ClassShaman: {
		// Forever: merged from SpellCrit 2.3000 + MeleeCrit 1.7000, max().
		stats.Crit:  2.3000 * CritRatingPerCritChance,
		stats.Dodge: 1.7000 * DodgeRatingPerDodgeChance,
	},
	proto.Class_ClassMage: {
		// Forever: merged from SpellCrit 0.2000 + MeleeCrit 3.2000. The melee
		// value is the Dodge column duplicated (see the note on the table),
		// so it is discarded and the spell value taken rather than max()'d.
		stats.Crit:  0.2000 * CritRatingPerCritChance,
		stats.Dodge: 3.2000 * DodgeRatingPerDodgeChance,
	},
	proto.Class_ClassWarlock: {
		// Forever: merged from SpellCrit 1.7000 + MeleeCrit 2.0000. The melee
		// value is the Dodge column duplicated (see the note on the table),
		// so it is discarded and the spell value taken rather than max()'d.
		stats.Crit:  1.7000 * CritRatingPerCritChance,
		stats.Dodge: 2.0000 * DodgeRatingPerDodgeChance,
	},
	proto.Class_ClassDruid: {
		// Forever: merged from SpellCrit 1.8000 + MeleeCrit 0.9000, max().
		stats.Crit:  1.8000 * CritRatingPerCritChance,
		stats.Dodge: 0.9000 * DodgeRatingPerDodgeChance,
	},
}

var APPerStrength = map[proto.Class]float64{
	proto.Class_ClassWarrior: 2,
	proto.Class_ClassPaladin: 2,
	proto.Class_ClassHunter:  1,
	proto.Class_ClassRogue:   1,
	proto.Class_ClassPriest:  1,
	proto.Class_ClassShaman:  2,
	proto.Class_ClassMage:    1,
	proto.Class_ClassWarlock: 1,
	proto.Class_ClassDruid:   2,
}

var APPerAgility = map[proto.Class]float64{
	proto.Class_ClassWarrior: 0,
	proto.Class_ClassPaladin: 0,
	proto.Class_ClassHunter:  0,
	proto.Class_ClassRogue:   1,
	proto.Class_ClassPriest:  0,
	proto.Class_ClassShaman:  0,
	proto.Class_ClassMage:    0,
	proto.Class_ClassWarlock: 0,
	proto.Class_ClassDruid:   1,
}

// CritPerAgiAtLevel is the percent of crit chance one point of Agility
// grants class at level, from the client's PlayerExpectedStat table
// (base_stats_levels_auto_gen.go's critPerAgiByClassLevel). The rate is
// higher at low levels (a level-30 warrior needs about 10 Agility per 1%,
// a level-60 one 20). A class absent from the generated table
// (ClassUnknown) grants none at any level.
func CritPerAgiAtLevel(class proto.Class, level int32) float64 {
	level = EffectiveCharacterLevel(level)
	if perLevel, ok := critPerAgiByClassLevel[class]; ok {
		return perLevel[level]
	}
	return 0
}

// Dodge agility scaling
var DodgePerAgiAtLevel = map[proto.Class]float64{
	proto.Class_ClassUnknown: 0.0,
	proto.Class_ClassWarrior: 0.0500,
	proto.Class_ClassPaladin: 0.0506,
	proto.Class_ClassHunter:  0.0378,
	proto.Class_ClassRogue:   0.0690,
	proto.Class_ClassPriest:  0.0500,
	proto.Class_ClassShaman:  0.0508,
	proto.Class_ClassMage:    0.0514,
	proto.Class_ClassWarlock: 0.0500,
	proto.Class_ClassDruid:   0.0500,
}

// AddCritStatDependencies wires character's Crit stat to both Agility and
// Intellect for every class, at the client's per-level rates
// (CritPerAgiAtLevel and SpellCritPerIntAtLevel at character.Level). A
// class whose rate is 0 at a level simply adds 0: Warrior and Rogue have
// no Intellect rate at any level. Pet units borrow another class's rate
// and are wired directly at their own call sites.
//
// unconfirmed: Forever's one Crit stat is confirmed by Blizzard, but
// whether a hybrid's Agility crit and Intellect crit both land in it, or
// whether the game keeps two pools and unifies only item, talent and
// racial crit, is unmeasured. The beta character sheet test (a hybrid with
// an Agility item on and off, watching whether the one Crit number moves)
// decides it. The magnitude at stake: Grace of Air's 77 Agility moved a
// balance druid's Starfire crit 3.4 points in this model.
func AddCritStatDependencies(character *Character, class proto.Class) {
	character.AddStatDependency(stats.Agility, stats.Crit, CritPerAgiAtLevel(class, character.Level)*CritRatingPerCritChance)
	character.AddStatDependency(stats.Intellect, stats.Crit, SpellCritPerIntAtLevel(class, character.Level)*CritRatingPerCritChance)
}

// allClasses lists every class the per-level tables (and the classAttack
// PowerOffsetAtLevel switch below) carry a row for. ClassUnknown is
// deliberately excluded: it stays the zero Stats{} at every level, the
// same as before this table existed.
var allClasses = []proto.Class{
	proto.Class_ClassWarrior,
	proto.Class_ClassPaladin,
	proto.Class_ClassHunter,
	proto.Class_ClassRogue,
	proto.Class_ClassPriest,
	proto.Class_ClassShaman,
	proto.Class_ClassMage,
	proto.Class_ClassWarlock,
	proto.Class_ClassDruid,
}

// BaseStatsAtLevel returns the six base stats (Health, Mana, Agility,
// Strength, Intellect, Spirit, Stamina) wowhead's gear planner reports for
// class at level (base_stats_levels_auto_gen.go), without race offsets,
// ClassBaseCrit, or Attack Power - getBaseStatsCombo adds those. A class
// absent from the generated table (ClassUnknown) returns the zero
// Stats{}.
func BaseStatsAtLevel(class proto.Class, level int32) stats.Stats {
	level = EffectiveCharacterLevel(level)
	if perLevel, ok := baseStatsByClassLevel[class]; ok {
		return perLevel[level]
	}
	return stats.Stats{}
}

// SpellCritPerIntAtLevel is the percent of spell crit chance one point of
// Intellect grants class at level, from the client's PlayerExpectedStat
// table (base_stats_levels_auto_gen.go's spellCritPerIntByClassLevel). A
// class absent from the generated table (Warrior, Rogue, ClassUnknown)
// grants none at any level.
func SpellCritPerIntAtLevel(class proto.Class, level int32) float64 {
	level = EffectiveCharacterLevel(level)
	if perLevel, ok := spellCritPerIntByClassLevel[class]; ok {
		return perLevel[level]
	}
	return 0
}

// classAttackPowerOffsetAtLevel is the level-dependent term ClassBaseStats'
// old AttackPower field baked in at level 60 only. Warrior, Paladin,
// Hunter, Rogue and Shaman wrote that field as a visible multiplication
// by 60 (60*3-20, 60*2-20) - the Classic ruleset's own formula, so it is
// expressed here as a function of level rather than re-derived. Priest,
// Mage, Warlock and Druid wrote theirs as bare literals (-10, -10, -10,
// -20) with no such multiplication, so nothing is invented for them: they
// keep the same flat value at every level, exactly as they did when
// CharacterMaxLevel was the only level that existed.
func classAttackPowerOffsetAtLevel(class proto.Class, level int32) float64 {
	level = EffectiveCharacterLevel(level)
	switch class {
	case proto.Class_ClassWarrior, proto.Class_ClassPaladin:
		return float64(level)*3 - 20
	case proto.Class_ClassHunter, proto.Class_ClassRogue, proto.Class_ClassShaman:
		return float64(level)*2 - 20
	case proto.Class_ClassPriest, proto.Class_ClassMage, proto.Class_ClassWarlock:
		return -10
	case proto.Class_ClassDruid:
		return -20
	default:
		return 0
	}
}

// BaseAttackPowerAtLevel is classAttackPowerOffsetAtLevel exported for
// class packages that need a level-aware base Attack Power outside a full
// getBaseStatsCombo call.
func BaseAttackPowerAtLevel(class proto.Class, level int32) float64 {
	return classAttackPowerOffsetAtLevel(class, level)
}

// classBaseStatsAtLevel composes one class's full base-stat row at level:
// the six stats wowhead's gear planner reports per level
// (BaseStatsAtLevel; base_stats_levels_auto_gen.go) plus the Attack Power
// (and, for Hunter, Ranged Attack Power) offset for that level. It does
// not add race offsets or ClassBaseCrit - getBaseStatsCombo does that.
func classBaseStatsAtLevel(class proto.Class, level int32) stats.Stats {
	row := BaseStatsAtLevel(class, level)
	row[stats.AttackPower] = classAttackPowerOffsetAtLevel(class, level)
	if class == proto.Class_ClassHunter {
		row[stats.RangedAttackPower] = classAttackPowerOffsetAtLevel(class, level)
	}
	return row
}

// ClassBaseStats is classBaseStatsAtLevel at CharacterMaxLevel, computed
// rather than hand-typed: base_stats_test.go's
// TestGeneratedLevel60MatchesTheOldClassBaseStats pins it against the
// literal values this map held before base_stats_levels_auto_gen.go
// existed, so a generator regression is caught rather than silently
// changing every level-60 sim.
var ClassBaseStats = buildClassBaseStats()

func buildClassBaseStats() map[proto.Class]stats.Stats {
	out := map[proto.Class]stats.Stats{
		proto.Class_ClassUnknown: {},
	}
	for _, class := range allClasses {
		out[class] = classBaseStatsAtLevel(class, CharacterMaxLevel)
	}
	return out
}

// getBaseStatsCombo retrieves base stats at level, with race offsets and
// crit rating adjustments. Race offsets are unaffected by level: wowhead's
// gear planner carries only one raceOffsets table (not one per level), and
// research/08-stats.md names no per-level racial drift, so RaceOffsets
// stays exactly as it was before per-level base stats existed.
func getBaseStatsCombo(r proto.Race, c proto.Class, level int32) stats.Stats {
	starting := classBaseStatsAtLevel(c, level)
	return starting.Add(RaceOffsets[r]).Add(ClassBaseCrit[c])
}
