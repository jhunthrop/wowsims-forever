#!/usr/bin/env python3
"""Generate sim/<class>/relic_mods_auto_gen.go: the equip-effect spell
modifiers of every paladin libram, shaman totem and druid idol.

A relic states its value as an equip spell (ItemEffect.TriggerType 1) whose
effect is a spell-family modifier. This script reads that chain from the
client's raw CSVs and writes each modifier as a core.EquipSpellMod, keeping
the client spell id, the spell text and the raw class mask beside it:

    ItemXItemEffect.csv  ItemID -> ItemEffectID
    ItemEffect.csv       TriggerType, SpellID
    SpellEffect.csv      EffectAura, EffectBasePointsF, EffectMiscValue_0,
                         EffectSpellClassMask_0..3
    Spell.csv            Description_lang (the comment text)
    SpellName.csv        Name_lang

Only auras 107 (ADD_FLAT_MODIFIER) and 108 (ADD_PCT_MODIFIER) are tables
here. A relic whose equip spell is anything else (a dummy, a proc trigger,
the vanilla class-script aura 112) is named in the file's trailing comment
and left to a hand-written effect; so is every relic below item level 40.
The table records every modifier faithfully, including ones the engine
cannot express yet; which items the engine actually claims is the explicit
core.NewEquipModItemEffect list in each class's items.go.

Usage (from the fork root):
    python3 tools/relicmods/gen.py <site repo>/data/builds/<build> [--check]

Run it with `python3 -I` from a directory that holds no other python files.
"""
from __future__ import annotations

import collections
import csv
import json
import sys
from pathlib import Path

MIN_ITEM_LEVEL = 40
TRIGGER_ON_EQUIP = 1
MODIFIER_AURAS = {107, 108}
RELIC_CLASSES = ("paladin", "shaman", "druid")
CLASS_SET = {"paladin": 10, "shaman": 11, "druid": 7}  # SpellClassSet, used only for the header
FORK_ROOT = Path(__file__).resolve().parent.parent.parent


def read_csv(raw: Path, name: str) -> list[dict[str, str]]:
    with open(raw / name, encoding="utf-8", newline="") as handle:
        return list(csv.DictReader(handle))


def u32(value: str) -> int:
    return int(value) & 0xFFFFFFFF


def one_line(text: str) -> str:
    return " ".join(text.replace("\r", " ").replace("\n", " ").split())


class Client:
    def __init__(self, raw: Path) -> None:
        self.effects_by_item: dict[int, list[tuple[int, int]]] = collections.defaultdict(list)
        item_effects = {int(r["ID"]): r for r in read_csv(raw, "ItemEffect.csv")}
        for link in read_csv(raw, "ItemXItemEffect.csv"):
            row = item_effects.get(int(link["ItemEffectID"]))
            if row is not None:
                self.effects_by_item[int(link["ItemID"])].append((int(row["TriggerType"]), int(row["SpellID"])))
        self.spell_effects: dict[int, list[dict[str, str]]] = collections.defaultdict(list)
        for row in read_csv(raw, "SpellEffect.csv"):
            if row.get("DifficultyID", "0") in ("0", ""):
                self.spell_effects[int(row["SpellID"])].append(row)
        self.names = {int(r["ID"]): r["Name_lang"] for r in read_csv(raw, "SpellName.csv")}
        self.text = {int(r["ID"]): r["Description_lang"] for r in read_csv(raw, "Spell.csv")}

    def equip_spells(self, item_id: int) -> list[int]:
        return [spell for trigger, spell in self.effects_by_item.get(item_id, []) if trigger == TRIGGER_ON_EQUIP]


def relics(items_dir: Path, class_slug: str) -> list[dict]:
    items = json.loads((items_dir / f"{class_slug}.json").read_text(encoding="utf-8"))["items"]
    return sorted(
        (i for i in items if i["slot"] == "ranged" and i["dps"] == 0 and i["item_level"] >= MIN_ITEM_LEVEL),
        key=lambda i: i["id"],
    )


def render_class(class_slug: str, build: str, client: Client, items: list[dict]) -> str:
    tabled: list[str] = []
    skipped: list[str] = []
    for item in items:
        mods: list[str] = []
        spell_notes: list[str] = []
        for spell in client.equip_spells(item["id"]):
            rows = [r for r in client.spell_effects.get(spell, []) if int(r["EffectAura"]) in MODIFIER_AURAS]
            for row in rows:
                families = ", ".join(str(u32(row[f"EffectSpellClassMask_{n}"])) for n in range(4))
                mods.append(
                    f"{{ClientSpellID: {spell}, Aura: {row['EffectAura']}, Op: {row['EffectMiscValue_0']}, "
                    f"Amount: {int(float(row['EffectBasePointsF']))}, Families: core.ClientClassMask{{{families}}}}}"
                )
            if rows:
                spell_notes.append(f'spell {spell} "{client.names.get(spell, "?")}": {one_line(client.text.get(spell, ""))}')
        header = f'{item["id"]} {item["name"]} (item level {item["item_level"]})'
        if mods:
            tabled.append("\t// " + header + "\n" + "".join(f"\t// {note}\n" for note in spell_notes)
                          + f"\t{item['id']}: {{\n" + "".join(f"\t\t{mod},\n" for mod in mods) + "\t},\n")
        else:
            auras = sorted({int(r["EffectAura"]) for s in client.equip_spells(item["id"]) for r in client.spell_effects.get(s, [])})
            skipped.append(f"// {header}: equip spell auras {auras or 'none'}")
    body = "".join(tabled)
    skipped_block = "\n".join(skipped)
    return (
        "// Code generated by tools/relicmods/gen.py. DO NOT EDIT.\n"
        f"//\n// Class: {class_slug}\n// Client build: {build}\n"
        f"// Relics at item level {MIN_ITEM_LEVEL}+ whose equip spell is aura 107/108 (spell-family modifier).\n"
        "// Regenerate with `make relicmods`.\n\n"
        f"package {class_slug}\n\n"
        'import "github.com/wowsims/classic/sim/core"\n\n'
        "// relicEquipMods is every client equip modifier of this class's relics, keyed by item id.\n"
        "// Which of them the engine models is the explicit registration list in items.go.\n"
        f"var relicEquipMods = map[int32][]core.EquipSpellMod{{\n{body}}}\n\n"
        "// Relics whose equip spell is not a 107/108 modifier; any model is hand-written in items.go.\n"
        f"{skipped_block}\n"
    )


def main() -> int:
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    check = "--check" in sys.argv
    if len(args) != 1:
        print(__doc__, file=sys.stderr)
        return 2
    build_dir = Path(args[0]).resolve()
    client = Client(build_dir / "raw")
    stale = False
    for class_slug in RELIC_CLASSES:
        text = render_class(class_slug, build_dir.name, client, relics(build_dir / "items", class_slug))
        out = FORK_ROOT / "sim" / class_slug / "relic_mods_auto_gen.go"
        if check:
            stale = stale or not out.exists() or out.read_text(encoding="utf-8") != text
        else:
            out.write_text(text, encoding="utf-8")
            print(f"wrote {out}")
    return 1 if stale else 0


if __name__ == "__main__":
    raise SystemExit(main())
