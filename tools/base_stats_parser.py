#!/usr/bin/env python3
"""Generate sim/core/base_stats_auto_gen.go from the client's GameTables.

Forever has no combat-rating system: research/08-stats.md 2 settles it as
"FLAT PERCENTAGES...the major engine decision and it is settled" (item
cards, talents and racials all read in percent; 470 datamined talents
never reference a rating). So every constant this generator emits is a
flat 1:1 percentage, never a rating-to-percent conversion factor - that
includes Defense/Dodge/Parry/Block, which earlier drafts of this
generator read out of combatratings.txt as if they were ratings. They are
not: combatratings.txt is present in the client (build 1.60.1.69893) but
is lineage boilerplate, not the rule - see PERCENTAGE_CONSTANTS below and
the header this script writes.

basemp.txt (per-class base mana) and hppersta.txt (health per point of
stamina) are two of the same three confirmed GameTables files, and their
build-1.60.1.69893 values genuinely do confirm two of the fork's existing
hand-typed values (ClassBaseStats' Mana fields and the Stamina->Health
dependency in character.go). This generator reads and records both, but
does not redefine either: sim/core/base_stats_test.go pins the match
instead, so a future mismatch is caught rather than silently ignored.

This script also generates sim/core/base_stats_levels_auto_gen.go from a
levels.json-shaped file: wowhead's Forever gear planner payload
(wow.gearPlanner.classic.baseStats and .critSpell), giving every class's
Health, Mana, Agility, Strength, Intellect, Spirit, Stamina and
spell-crit-per-Intellect rate at every level 1..60, not just 60. The site
repo's data lane emits this same content as builds/<build>/levels.json;
this fork vendors a copy (assets/db_inputs/levels/<build>.json) so the
generated file is reproducible from something checked into this repo.
Race offsets and Attack Power are not read from it: race offsets stay
base_stats.go's RaceOffsets (wowhead's raceOffsets table matches it,
checked by hand, not regenerated - see docs/superpowers/specs/
2026-09-27-level-aware-sim-design.md's lane report), and Attack Power has
no wowhead table at all - it is a function of level in base_stats.go.

Usage:
    python3 tools/base_stats_parser.py                                  # 1.60.1.69893, vendored copy
    python3 tools/base_stats_parser.py --inputs assets/db_inputs/gametables/1.60.2.70000
    python3 tools/base_stats_parser.py --build 1.60.2.70000 \\
        --inputs assets/db_inputs/gametables/1.60.2.70000
    python3 tools/base_stats_parser.py --levels-json assets/db_inputs/levels/1.60.1.69893.json
"""

import argparse
import csv
import json
import os
import sys

MAX_LEVEL = 60

# wowhead's gear planner classId -> proto.Class name (proto/api.proto's
# Class enum). Classes the client has no gear planner data for (Death
# Knight, Monk, Demon Hunter, Evoker: not in Classic) are absent from
# both sides and never appear in the generated tables.
LEVELS_CLASS_ID_TO_PROTO_NAME = {
    "1": "ClassWarrior",
    "2": "ClassPaladin",
    "3": "ClassHunter",
    "4": "ClassRogue",
    "5": "ClassPriest",
    "7": "ClassShaman",
    "8": "ClassMage",
    "9": "ClassWarlock",
    "11": "ClassDruid",
}

# wowhead's gear planner statId -> stats.Stat field name (sim/core/stats),
# in the order base_stats_levels_auto_gen.go writes each level's struct
# literal. statId 2 (Stamina in some wowhead tables) is not used by the
# baseStats table this fork reads; only these seven appear in it.
LEVELS_STAT_ID_TO_FIELD = [
    ("1", "Health"),
    ("0", "Mana"),
    ("3", "Agility"),
    ("4", "Strength"),
    ("5", "Intellect"),
    ("6", "Spirit"),
    ("7", "Stamina"),
]

COMBAT_RATINGS = "combatratings.txt"
BASE_MP = "basemp.txt"
HP_PER_STA = "hppersta.txt"

# Every constant this generator emits is a straight 1:1 percentage -
# research/08-stats.md 2 settles Forever as flat-percentage, full stop,
# and that includes the four stats combatratings.txt calls "ratings".
# Confirmed, not provisional (research/08-stats.md 2 and 12.4).
PERCENTAGE_CONSTANTS = [
    ("CritRatingPerCritChance", "Forever: one crit stat for spells and melee."),
    ("HitRatingPerHitChance", "Forever: one hit stat for spells, melee and ranged."),
    ("HasteRatingPerHastePercent", "Forever keeps melee, ranged and spell haste separate."),
    ("ExpertiseRatingPerExpertiseChance",
     "The item unit is a percentage (Edgemaster's 1.0%), not expertise points."),
    ("DefenseRatingPerDefense",
     "combatratings.txt carries a 'Defense Skill' column, but it is not the rule (see header)."),
    ("DodgeRatingPerDodgeChance",
     "combatratings.txt carries a 'Dodge' column, but it is not the rule (see header)."),
    ("ParryRatingPerParryChance",
     "combatratings.txt carries a 'Parry' column, but it is not the rule (see header)."),
    ("BlockRatingPerBlockChance",
     "combatratings.txt carries a 'Block' column, but it is not the rule (see header)."),
]

# Columns of combatratings.txt reported (never applied) in the generated
# header, to show exactly what the table says at level 60 and that it was
# actually read rather than dismissed unseen.
COMBAT_RATINGS_REPORT_COLUMNS = [
    "Defense Skill", "Dodge", "Parry", "Block", "Hit - Melee", "Crit - Melee",
]

# basemp.txt's per-class columns, in ClassBaseStats' declaration order
# (sim/core/base_stats.go), for the header's confirmation block.
BASE_MP_CLASSES = [
    "Warrior", "Paladin", "Hunter", "Rogue", "Priest",
    "Shaman", "Mage", "Warlock", "Druid",
]

# Tables that do not exist for build 1.60.1.69893 (404 on wago; not among
# the three GameTables files the data lane has mined) and what each would
# supply if it landed. Listed once in the generated header so a reader
# never has to guess why a given value is still hand-typed elsewhere.
ABSENT_TABLES = [
    ("chancetomeleecrit.txt / chancetomeleecritbase.txt",
     "per-class melee crit base values (kept as Era's in sim/core/base_stats.go's ClassBaseCrit)"),
    ("chancetospellcrit.txt / chancetospellcritbase.txt",
     "per-class spell crit base values (same table; dropped from ClassBaseCrit by the Hit/Crit merge)"),
    ("octbasempbyclass.txt / octclasscombatratingscalar.txt",
     "an Era-era alternate mana/rating-scalar encoding, superseded by basemp.txt above"),
    ("a per-race/per-class basestats/ mining directory",
     "Str/Agi/Sta/Int/Spirit/Health/AttackPower allocations (kept as Era's in ClassBaseStats/RaceOffsets)"),
]


def read_column_indexed(path):
    """A GameTables file: one row per level with named columns. Return
    {column name: [value per level]}."""
    with open(path, newline="") as fh:
        rows = list(csv.reader(fh, delimiter="\t"))
    if not rows:
        raise SystemExit(f"{path} is empty")
    header = [h.strip() for h in rows[0]]
    out = {}
    for col_idx, name in enumerate(header):
        if not name or name == "Level":
            continue
        values = []
        for row in rows[1:]:
            if col_idx < len(row) and row[col_idx].strip():
                values.append(float(row[col_idx]))
        if name in out:
            raise SystemExit(f"{path} has the column {name!r} more than once")
        out[name] = values
    return out


def at_max_level(table, column, path):
    if column not in table:
        raise SystemExit(
            f"{path} has no column {column!r}; columns are {sorted(table)}")
    values = table[column]
    if len(values) < MAX_LEVEL:
        raise SystemExit(
            f"{path} column {column!r} has {len(values)} rows, need at least {MAX_LEVEL}")
    return values[MAX_LEVEL - 1]


def generate(build, inputs_path, combat_ratings, base_mp, hp_per_sta):
    lines = [
        "// Code generated by tools/base_stats_parser.py. DO NOT EDIT.",
        "//",
        f"// Client build: {build}",
        f"// Source:       {inputs_path}/{{{COMBAT_RATINGS},{BASE_MP},{HP_PER_STA}}}",
        "//",
        "// Regenerate with:",
        f"//     python3 tools/base_stats_parser.py --build {build} --inputs {inputs_path}",
        "",
        "package core",
        "",
        "// BaseStatsBuild is the client build these constants were read from.",
        f'const BaseStatsBuild = "{build}"',
        "",
        "// combatratings.txt is present in this build and IS read (see the",
        "// values below), but is not the rule: research/08-stats.md 2 settles",
        "// Forever's rules as flat percentages, full stop, and this table's",
        "// entire purpose is rating-to-percent conversion. Its own columns",
        "// carry the tell: every one of the 123 level rows is identical (a",
        "// live rating table scales with level; this one is a level-invariant",
        "// stub), and it sits next to Mastery/Versatility/Corruption/PvP-Power",
        "// columns and the crit-taken-reduction stat Task 4 deleted from the",
        "// enum - none of which Forever has either. So every value below is",
        "// read from the table and then NOT applied - the constant stays 1.",
        f"// {COMBAT_RATINGS} level {MAX_LEVEL} (read, not applied):",
    ]
    for col in COMBAT_RATINGS_REPORT_COLUMNS:
        value = at_max_level(combat_ratings, col, f"{inputs_path}/{COMBAT_RATINGS}")
        lines.append(f"//   {col} = {value:g}")
    lines.append("//")
    lines.append(f"// {BASE_MP} and {HP_PER_STA} ARE two of the same three confirmed")
    lines.append("// GameTables files, and their values genuinely confirm two of the")
    lines.append("// fork's existing hand-typed numbers - they are not redefined here")
    lines.append("// (that stays in sim/core/base_stats.go's ClassBaseStats and")
    lines.append("// sim/core/character.go's Stamina->Health dependency, to avoid a")
    lines.append("// second source of truth), but the match is pinned by")
    lines.append("// sim/core/base_stats_test.go so a future mismatch is caught.")
    lines.append(f"// source build: {build} ({BASE_MP} level {MAX_LEVEL} Mana, confirms sim/core/base_stats.go's ClassBaseStats):")
    for cls in BASE_MP_CLASSES:
        value = at_max_level(base_mp, cls, f"{inputs_path}/{BASE_MP}")
        lines.append(f"//   {cls} = {value:g}")
    hp_value = at_max_level(hp_per_sta, "Health", f"{inputs_path}/{HP_PER_STA}")
    lines.append(f"// source build: {build} ({HP_PER_STA} level {MAX_LEVEL} Health-per-Stamina, confirms sim/core/character.go:283):")
    lines.append(f"//   Health = {hp_value:g}")
    lines.append("//")
    lines.append("// Absent tables (do not exist for build 1.60.1.69893; not among the")
    lines.append("// three confirmed GameTables files) and what each would supply:")
    for name, supplies in ABSENT_TABLES:
        lines.append(f"//   {name}")
        lines.append(f"//     -> {supplies}")
    lines.append("")

    for name, comment in PERCENTAGE_CONSTANTS:
        lines.append(f"// {comment}")
        lines.append("// Ratings are straight percentages in the Classic lineage.")
        lines.append(f"const {name} = 1")
    lines.append("")
    return "\n".join(lines)


def read_levels_json(path):
    """A levels.json-shaped file (see the module docstring). Return
    (base_stats, crit_spell) where base_stats[className][level] is a dict
    of field name -> value and crit_spell[className][level] is a float,
    for level in 1..MAX_LEVEL."""
    with open(path) as fh:
        data = json.load(fh)

    stats_by_class_id = data["baseStats"]["stats"]
    base_stats = {}
    for class_id, class_name in LEVELS_CLASS_ID_TO_PROTO_NAME.items():
        if class_id not in stats_by_class_id:
            continue
        by_stat_id = stats_by_class_id[class_id]
        by_level = {}
        for level in range(1, MAX_LEVEL + 1):
            row = {}
            for stat_id, field in LEVELS_STAT_ID_TO_FIELD:
                values = by_stat_id.get(stat_id)
                if values is None or level >= len(values):
                    raise SystemExit(
                        f"{path}: baseStats.stats[{class_id}][{stat_id}] "
                        f"({class_name}) has no level {level} entry")
                row[field] = values[level]
            by_level[level] = row
        base_stats[class_name] = by_level

    crit_spell_by_class_id = data.get("critSpell", {})
    crit_spell = {}
    for class_id, class_name in LEVELS_CLASS_ID_TO_PROTO_NAME.items():
        values = crit_spell_by_class_id.get(class_id)
        if values is None:
            continue
        by_level = {}
        for level in range(1, MAX_LEVEL + 1):
            if level >= len(values):
                raise SystemExit(
                    f"{path}: critSpell[{class_id}] ({class_name}) has no "
                    f"level {level} entry")
            by_level[level] = values[level]
        crit_spell[class_name] = by_level

    return base_stats, crit_spell


def go_number(value):
    """A JSON number formatted as a Go literal: Python's float repr is
    already the shortest string that round-trips, so a source value like
    0.075 is never turned into 0.07500000000000001 or similar."""
    if isinstance(value, float) and value.is_integer():
        return repr(int(value))
    return repr(value)


def generate_levels(source_path, base_stats, crit_spell):
    lines = [
        "// Code generated by tools/base_stats_parser.py. DO NOT EDIT.",
        "//",
        f"// Source: {source_path} (wowhead's Forever gear planner payload:",
        "// wow.gearPlanner.classic.baseStats and .critSpell)",
        "//",
        "// Regenerate with:",
        f"//     python3 tools/base_stats_parser.py --levels-json {source_path}",
        "",
        "package core",
        "",
        "import (",
        '\t"github.com/wowsims/classic/sim/core/proto"',
        '\t"github.com/wowsims/classic/sim/core/stats"',
        ")",
        "",
        "// LevelStatsSource names the wowhead gear-planner payload",
        "// baseStatsByClassLevel and spellCritPerIntByClassLevel were read from.",
        f'const LevelStatsSource = "{source_path} (wow.gearPlanner.classic.baseStats, .critSpell)"',
        "",
        "// baseStatsByClassLevel[class][level] is wowhead's gear planner Health,",
        "// Mana, Agility, Strength, Intellect, Spirit and Stamina for that class at",
        "// that level, level 1..CharacterMaxLevel (index 0 is unused, the zero",
        "// Stats{}, matching the source array's own unused index 0). Level",
        "// CharacterMaxLevel reproduces base_stats.go's ClassBaseStats exactly for",
        "// every class here - TestGeneratedLevel60MatchesTheOldClassBaseStats pins",
        "// it. Attack Power is not part of this table: wowhead's gear planner does",
        "// not carry it, and base_stats.go derives it from level instead",
        "// (classAttackPowerOffsetAtLevel).",
        "var baseStatsByClassLevel = map[proto.Class][CharacterMaxLevel + 1]stats.Stats{",
    ]
    for class_name, by_level in base_stats.items():
        lines.append(f"\tproto.Class_{class_name}: {{")
        for level in range(1, MAX_LEVEL + 1):
            row = by_level[level]
            fields = ", ".join(
                f"stats.{field}: {go_number(row[field])}"
                for _, field in LEVELS_STAT_ID_TO_FIELD
                if row[field] != 0
            )
            lines.append(f"\t\t{level}: {{{fields}}},")
        lines.append("\t},")
    lines.append("}")
    lines.append("")

    lines.append("// spellCritPerIntByClassLevel[class][level] is the fraction of spell crit")
    lines.append("// chance one point of Intellect grants at that level (same unit as the")
    lines.append("// pre-generator flat CritPerIntAtLevel), level 1..CharacterMaxLevel. A")
    lines.append("// class absent here (Warrior, Rogue) grants none at any level -")
    lines.append("// SpellCritPerIntAtLevel returns 0 for it, matching CritPerIntAtLevel's")
    lines.append("// old 0.0 rows for those two classes.")
    lines.append("var spellCritPerIntByClassLevel = map[proto.Class][CharacterMaxLevel + 1]float64{")
    for class_name, by_level in crit_spell.items():
        values = ", ".join(f"{level}: {go_number(by_level[level])}" for level in range(1, MAX_LEVEL + 1))
        lines.append(f"\tproto.Class_{class_name}: {{{values}}},")
    lines.append("}")
    lines.append("")
    return "\n".join(lines)


def resolve_build(build, inputs_path):
    if build:
        return build
    inferred = os.path.basename(os.path.normpath(inputs_path))
    if inferred and inferred not in (".", os.sep):
        return inferred
    build_file = os.path.join(inputs_path, "BUILD")
    if os.path.exists(build_file):
        return open(build_file).read().strip()
    raise SystemExit(
        "no --build given, the --inputs directory name isn't usable as a "
        "build string, and no BUILD file exists in --inputs; the generated "
        "constants must record which client they came from")


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--inputs", default="assets/db_inputs/gametables/1.60.1.69893",
                     help="directory holding the client's GameTables files")
    ap.add_argument("--build", default="",
                     help="client build string, e.g. 1.60.1.69893; inferred from "
                          "--inputs' basename, or an --inputs/BUILD file, when omitted")
    ap.add_argument("--out", default="sim/core/base_stats_auto_gen.go")
    ap.add_argument("--levels-json", default="assets/db_inputs/levels/1.60.1.69893.json",
                     help="wowhead gear-planner payload (levels.json shape); "
                          "pass an empty string to skip this file's generation")
    ap.add_argument("--levels-out", default="sim/core/base_stats_levels_auto_gen.go")
    args = ap.parse_args()

    build = resolve_build(args.build, args.inputs)

    combat_ratings = read_column_indexed(f"{args.inputs}/{COMBAT_RATINGS}")
    base_mp = read_column_indexed(f"{args.inputs}/{BASE_MP}")
    hp_per_sta = read_column_indexed(f"{args.inputs}/{HP_PER_STA}")

    out = generate(build, args.inputs, combat_ratings, base_mp, hp_per_sta)
    with open(args.out, "w") as fh:
        fh.write(out)
    print(f"wrote {args.out} from build {build}", file=sys.stderr)

    if args.levels_json:
        base_stats, crit_spell = read_levels_json(args.levels_json)
        levels_out = generate_levels(args.levels_json, base_stats, crit_spell)
        with open(args.levels_out, "w") as fh:
            fh.write(levels_out)
        print(f"wrote {args.levels_out} from {args.levels_json}", file=sys.stderr)


if __name__ == "__main__":
    main()
