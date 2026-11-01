#!/usr/bin/env python3
"""Generate sim/core/eureka_auto_gen.go: each class's Eureka! row and the
spells its effects cover.

Eureka! (Gnome) is one client spell per class, told apart by
SpellClassOptions.SpellClassSet. Each row's aura-108 effects (add-percent
spell modifiers) name a modifier in EffectMiscValue_0 (0 damage or healing,
22 periodic damage, 14 cost) and the spells it reaches in
EffectSpellClassMask_0..3, tested against each spell's own SpellClassMask
in the same class set. This lists, per class and modifier, the percent and
every spell id the mask reaches, plus the row's duration and cooldown.

    SpellName.csv          Name_lang ("Eureka!")
    SpellClassOptions.csv  SpellClassSet, SpellClassMask_0..3
    SpellEffect.csv        Effect, EffectAura, EffectMiscValue_0,
                           EffectBasePointsF, EffectSpellClassMask_0..3
    SpellMisc.csv          DurationIndex
    SpellDuration.csv      Duration
    SpellCooldowns.csv     RecoveryTime

Usage (from the fork root):
    python3 -I tools/eureka/gen.py <site repo>/data/builds/<build> [--check]

Run it with `python3 -I` from a directory that holds no other python files.
"""
from __future__ import annotations

import csv
import subprocess
import sys
from pathlib import Path

FORK_ROOT = Path(__file__).resolve().parent.parent.parent
OUT = FORK_ROOT / "sim" / "core" / "eureka_auto_gen.go"
NAME = "Eureka!"
APPLY_AURA = "6"
ADD_PCT_MODIFIER = "108"
MODIFIERS = {"0": "Damage", "22": "Periodic", "14": "Cost"}
# SpellClassSet (SpellFamilyName) to the engine's proto class.
CLASS_SETS = {
    "3": "proto.Class_ClassMage",
    "4": "proto.Class_ClassWarrior",
    "5": "proto.Class_ClassWarlock",
    "6": "proto.Class_ClassPriest",
    "7": "proto.Class_ClassDruid",
    "8": "proto.Class_ClassRogue",
    "9": "proto.Class_ClassHunter",
    "10": "proto.Class_ClassPaladin",
    "11": "proto.Class_ClassShaman",
}


def read_csv(raw: Path, name: str) -> list[dict[str, str]]:
    with open(raw / name, encoding="utf-8", newline="") as handle:
        return list(csv.DictReader(handle))


def mask_of(row: dict[str, str], prefix: str) -> tuple[int, ...]:
    return tuple(int(row[f"{prefix}{i}"]) & 0xFFFFFFFF for i in range(4))


def intersects(a: tuple[int, ...], b: tuple[int, ...]) -> bool:
    return any(x & y for x, y in zip(a, b))


def variants(raw: Path) -> list[dict]:
    names = {r["ID"]: r["Name_lang"] for r in read_csv(raw, "SpellName.csv")}
    options = {r["SpellID"]: r for r in read_csv(raw, "SpellClassOptions.csv")}
    effects: dict[str, list[dict[str, str]]] = {}
    for row in read_csv(raw, "SpellEffect.csv"):
        if row.get("DifficultyID", "0") in ("0", ""):
            effects.setdefault(row["SpellID"], []).append(row)
    durations = {r["ID"]: int(r["Duration"]) for r in read_csv(raw, "SpellDuration.csv")}
    misc = {r["SpellID"]: r for r in read_csv(raw, "SpellMisc.csv") if r.get("DifficultyID", "0") in ("0", "")}
    cooldowns = {r["SpellID"]: int(r["RecoveryTime"]) for r in read_csv(raw, "SpellCooldowns.csv")}

    out = []
    for spell_id, name in names.items():
        if name != NAME or spell_id not in options:
            continue
        class_set = options[spell_id]["SpellClassSet"]
        if class_set not in CLASS_SETS:
            continue
        family = {sid: mask_of(r, "SpellClassMask_") for sid, r in options.items() if r["SpellClassSet"] == class_set}
        mods = {}
        for eff in effects.get(spell_id, []):
            kind = MODIFIERS.get(eff["EffectMiscValue_0"])
            if eff["Effect"] != APPLY_AURA or eff["EffectAura"] != ADD_PCT_MODIFIER or kind is None:
                continue
            mask = mask_of(eff, "EffectSpellClassMask_")
            covered = sorted(int(sid) for sid, m in family.items() if intersects(m, mask))
            mods[kind] = (int(float(eff["EffectBasePointsF"])), covered)
        out.append({
            "spell_id": int(spell_id),
            "class": CLASS_SETS[class_set],
            "duration_ms": durations[misc[spell_id]["DurationIndex"]],
            "cooldown_ms": cooldowns.get(spell_id, 0),
            "mods": mods,
            "names": names,
        })
    return sorted(out, key=lambda v: v["spell_id"])


def render_set(kind: str, mod, names) -> str:
    if mod is None:
        return f"\t\t{kind}: eurekaModifier{{}},\n"
    percent, ids = mod
    rows = "".join(f"\t\t\t{i}: {{}}, // {names.get(str(i), '')}\n" for i in ids)
    return f"\t\t{kind}: eurekaModifier{{Percent: {percent}, Spells: map[int32]struct{{}}{{\n{rows}\t\t}}}},\n"


def render(build: str, found: list[dict]) -> str:
    body = ""
    for v in found:
        body += f"\t{v['class']}: {{\n"
        body += f"\t\tSpellID:    {v['spell_id']},\n"
        body += f"\t\tDurationMS: {v['duration_ms']},\n"
        body += f"\t\tCooldownMS: {v['cooldown_ms']},\n"
        for kind in ("Damage", "Periodic", "Cost"):
            body += render_set(kind, v["mods"].get(kind), v["names"])
        body += "\t},\n"
    return (
        "// Code generated by tools/eureka/gen.py. DO NOT EDIT.\n"
        "//\n"
        f"// Client build: {build}\n"
        "//\n"
        "// Each class's Eureka! row (one per SpellClassSet), its duration and\n"
        "// cooldown, and per add-percent modifier (EffectMiscValue_0 0 damage or\n"
        "// healing, 22 periodic damage, 14 cost) the percent and every spell id\n"
        "// whose SpellClassMask meets the effect's mask in the same class set.\n"
        "// Regenerate with `make eureka`.\n\n"
        "package core\n\n"
        "import \"github.com/wowsims/classic/sim/core/proto\"\n\n"
        "var eurekaByClass = map[proto.Class]eurekaVariant{\n"
        f"{body}"
        "}\n"
    )


def gofmt(text: str) -> str:
    """Align the literal the way gofmt does, so the file is gofmt-clean."""
    return subprocess.run(["gofmt"], input=text, capture_output=True, text=True, check=True).stdout


def main() -> int:
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    if len(args) != 1:
        print(__doc__, file=sys.stderr)
        return 2
    build_dir = Path(args[0]).resolve()
    found = variants(build_dir / "raw")
    if not found:
        print("no Eureka! rows found", file=sys.stderr)
        return 1
    text = gofmt(render(build_dir.name, found))
    if "--check" in sys.argv:
        return 0 if OUT.exists() and OUT.read_text(encoding="utf-8") == text else 1
    OUT.write_text(text, encoding="utf-8")
    print(f"wrote {OUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
