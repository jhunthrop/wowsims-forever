"""Render the site repo's data/builds/<build>/levels.json (the wowhead
gear-planner payload, flattened per class and level) back into the
payload shape tools/base_stats_parser.py reads: baseStats.stats[classId]
[statId][level] and critSpell[classId][level].

    python3 -I tools/levels_from_site.py <site levels.json> <out.json>

Lossless for the fields the parser reads: converting the 69893 site file
reproduced the vendored 69893 payload's stats and critSpell exactly.
"""
import json
import sys

CLASS_IDS = {
    "warrior": "1", "paladin": "2", "hunter": "3", "rogue": "4", "priest": "5",
    "shaman": "7", "mage": "8", "warlock": "9", "druid": "11",
}
STAT_IDS = {
    "mana": "0", "health": "1", "agility": "3", "strength": "4",
    "intellect": "5", "spirit": "6", "stamina": "7",
}


def render(site):
    stats, crit_spell = {}, {}
    for class_name, class_id in CLASS_IDS.items():
        rows = site["classes"][class_name]["levels"]
        by_stat = {stat_id: [0] for stat_id in STAT_IDS.values()}
        crit = [0]
        for row in rows:
            for field, stat_id in STAT_IDS.items():
                by_stat[stat_id].append(row[field])
            crit.append(row["spell_crit_per_int"] or 0)
        stats[class_id] = by_stat
        if any(crit):
            crit_spell[class_id] = crit
    return {"baseStats": {"stats": stats}, "critSpell": crit_spell}


def main():
    src, out = sys.argv[1:3]
    with open(src) as fh:
        site = json.load(fh)
    with open(out, "w") as fh:
        json.dump(render(site), fh)


if __name__ == "__main__":
    main()
