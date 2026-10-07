# Damage conformance: paladin, warlock, shaman, druid

Lane `damage-hybrids`, fork branch `damage-hybrids` (from `forever` at 93dcddd19). Ability files and their tests only; no change to `sim/core`, `sim/conformance`, the generator, `constants_auto_gen.go`, talents or the other classes.

## What shipped

- `sim/common/clientdamage`: `Effect` carries one rank's client roll in the `spellconst.Spell.DamageRange` convention (centre at the spell's own level, `PerLevel` growth, `MaxLevel` cap, `Variance` as the whole width). `Range(level)` is what a spell declares as `ClientBaseDamage`; `Roll(sim, level)` draws a hit; `FromRoll` and `FromTable` build an `Effect` from a generated `<Spell>BaseDamage`, `PointsPerLevel`, `Level` and `MaxLevel` row.
- `sim/common/clientdamage/clientdamagetest`: `AssertTable` checks a whole table against the vendored client file at levels 1, 10, 20, 30, 38, 40, 50 and 60 (range and spell level and cap, plus the coefficient when the caller passes one). Used by one `spellconst_damage_test.go` per class.
- Hand tables and flat literals are gone from every spell below; the generator does not emit a table for a spell that already has a hand-written `<Spell>Ranks` (and for Exorcism and Holy Wrath it picks the wrong duplicate id), so those tables are cut from the client JSON by the engine ids and the tests pin them to it. Rake, Rip and Ferocious Bite are not hand-written in the generator's sense and read its rows through `FromTable`.
- Process note: the test files were written with the conversions and then confirmed against the report, not strictly before each conversion.

## Conformance movement at level 60 (`not declared` -> `declared, matches`)

| Class | Before (not declared) | Declared | Matching | Differing | Still not declared |
|---|---|---|---|---|---|
| Shaman | 123 | 123 | 123 | 0 | 0 |
| Druid | 61 | 61 | 61 | 0 | 0 |
| Paladin | 59 | 43 | 43 | 0 | 16 |
| Warlock | 104 | 102 | 102 | 0 | 2 |

Nothing is left at `declared, differs`. (Mid-pass the coefficient column showed 52 differs, all coefficients: see below.)

Not declared, and why:

- Paladin Seal of Righteousness proc (8 rows, ids 25713-25742): the engine models the proc as weapon swing speed times the seal's percent, as the brief asks to leave weapon-percent effects alone; the client row's 35/32/25/.../4 is not a base the engine rolls.
- Warlock Bane of Doom (id 603, 2 rows): the engine registers the Era Curse of Doom numbers (3200, coefficient 1.0) under id 603, whose client row is 1742 at coefficient 4.0; the client's 3200/1.0 are on a different spell, 449432. Which is intended is the owner's call; the rotations do not cast it.

Rows whose non-damage columns moved: the Rank column of Rake, Rip and Ferocious Bite now reads 1-5 instead of 0 (these configs set `Rank`); no verdict changed.

## Spells changed (top rank at level 60: old -> new roll, coefficient)

Shaman (talent multipliers and Lightning Overload untouched; every hit now draws its own roll):

| Spell | Old | New | Coefficient |
|---|---|---|---|
| Lightning Bolt 10 | 196 flat | 189.9-211.7 | 0.714 |
| Chain Lightning 4 | 123 flat | 119.2-133.2 per bounce | 0.571 |
| Earth Shock 7 | 301 flat | 293.1-308.9 | 0.386 |
| Frost Shock 4 | 283 flat | 278.6-294.6 | 0.386 |
| Flame Shock 6 | 166 direct, 44 tick | 166 direct (growth capped at 15 levels above its own), 44 tick | 0.214 / 0.1 |
| Lava Burst 3 | 220 flat | 192.1-247.9 | 0.714 |
| Fire Nova 5 | 419 flat | 412.1-459.9 (centre 436, +3.4 a level to 57) | 0.143 |
| Magma Totem 4 | 73 flat | 73 flat | 0.033 |
| Searing Totem bolt 6 | 47 flat | 40.0-54.0 | 0.017 |

Druid:

| Spell | Old | New | Coefficient |
|---|---|---|---|
| Wrath 8 | 91 flat | 91.6-102.4 | 0.571 |
| Moonfire 10 | 135 flat, tick 60 | 128.7-150.5, tick 60 | 0.15 / 0.13 |
| Starfire 7 | 381 flat | 350.0-412.0 | 1.0 |
| Insect Swarm 5 | tick 31 | tick 31 | 0.158 (now also on the spell) |
| Rake 4 | 58 initial, 32 tick | 61 initial, 34 tick | none stated |
| Rip 6 | 17 tick base | 15 tick base | none stated |
| Ferocious Bite 5 | base 52 + 60 x rand | 52-112 from the generated row | 1.0 |

Paladin:

| Spell | Old | New | Coefficient |
|---|---|---|---|
| Hammer of Wrath 3 | 504-566 | 473.6-522.4 | 0.429 |
| Exorcism 6 | 505-563 | 474.7-529.3 | 0.429 |
| Holy Wrath 2 | 490-576 | 490.0-576.0 (now from the client table) | 0.19 |
| Judgement of Righteousness 8 | 170.2-186.2 (growth stopped at 60) | 169.8-186.6 | 0.5 on every rank (was 0.144/0.312/0.462 on ranks 1-3) |
| Judgement of Command 5 | 339-373 | 339.0-373.0 | 0.429 |
| Holy Shock 3 (cast 20930) | 365-395 | 334.3-361.7 (client spell 25902: 348, 0.0789 wide) | 0.429 |
| Holy Strike 8 flat bonus | 93 flat | 81.4-104.6 | 0.429 |

Warlock:

| Spell | Old | New | Coefficient |
|---|---|---|---|
| Shadow Bolt 10 | 268 flat | 253.3-282.7 | 0.857 (ranks 1-3 now 0.486/0.629/0.8, were 0.14/0.299/0.56) |
| Shadowburn 6 | 266 flat | 258.3-288.1 | 0.429 |
| Conflagrate 6 | 282 flat | 251.1-312.9 | 0.429 |
| Incinerate 3 | 217 flat | 200.7-233.3 | 0.714 |
| Searing Pain 6 | 114 flat | 107.0-125.8 | 0.429 (rank 1 was 0.396) |
| Soul Fire 2 | 431 flat | 389.3-487.9 | 1.0 |
| Immolate 8 | 158 direct, 55 tick | 158 direct, 55 tick | 0.2 direct (ranks 1-2 were 0.058/0.125), 0.13 tick (ranks 1-3 were 0.037/0.081) |
| Corruption 7 | 73 tick | 73 tick | 0.2 (was 0.167; ranks 1-2 were 0.08/0.155) |
| Bane of Agony 6 | 46 tick | 46 tick | 0.133 on every rank (was 0.046-0.083) |
| Drain Soul 4 | 84 tick | 84 tick | 0.1 (rank 1 was 0.063) |
| Drain Life 6 | 51 tick | 51 tick | 0.1 (rank 1 was 0.078) |
| Siphon Life 4 | 41 tick | 41 tick | 0.05 |
| Rain of Fire 4 | 226 tick | 221 tick at level 60 (220 at level 58, +0.6 a level), the client damage spells 1282380-1282385 | 0.083 |
| Death Coil 3 | 476 | 460 at level 60 (client 454 at level 58, +3 a level) | 0.214 |
| Wrack | 36 tick | 36 tick | 0.143 (now also on the spell) |

Pure DoT spells (Corruption, Bane of Agony, Drain Soul, Wrack, Insect Swarm) now also set the tick coefficient as the spell's `BonusCoefficient`. The conformance report reads only the spell's; a pure DoT never computes direct damage, so it has no effect on the sim.

Engine fixes on lines this lane rewrote: Immolate's `ExpectedTickDamage` (non-snapshot branch) passed the direct hit instead of the per-tick amount; Corruption's passed the per-tick amount divided by the tick count a second time. Both use the per-tick amount now.

## Goldens that moved

- Shaman: Elemental +0.16% to +0.55% DPS (mean +0.41%, 138 results), Enhancement -0.06% to +0.15% (mean +0.02%, 140 results), from the per-level growth (Lightning Bolt +4.8 at level 60, Chain Lightning +3.2) and the new rolls. Adopted in `sim/shaman/elemental/TestElemental.results` and `sim/shaman/enhancement/TestEnhancement.results`; the stat-weight rows moved in the third decimal only.
- Druid, paladin, warlock `.results`: no movement (the balance suite is skipped; the feral, retribution, protection and warlock suites did not move).
- Conformance goldens regenerated: `druid`, `paladin`, `shaman`, `warlock` and the damage block of `SUMMARY.md`; only the damage columns changed (and the Rank cells named above).

`go test --tags=with_db` over every `sim/...` package except `sim/web`: green except `TestCastingOnlyMovementIsFreeForTheFuryWarrior` in `sim`, which fails identically on 93dcddd19 (warrior, not touched here).

## Open

1. Bane of Doom / Curse of Doom: see above; needs an owner decision on which client spell the engine models.
2. Seal of Righteousness proc, Seal of Command proc, Judgement of Command's stun doubling, Stormstrike, Judgement: weapon-percent models, left alone; Consecration (client rows 1280345-1280349 and 20924 do not form a plain tick) and Seal of Righteousness' rank-0 rows are not compared.
3. Rip and Ferocious Bite per-combo-point and per-energy steps are still Era figures; the client's table does not carry them.
4. Holy Shock: the client has a rank 1 at level 30 (ids 1311604-1311606, 134 damage) that the engine does not register, and the damage numbers now follow the client's 25912/25911/25902 (about 12% under the Classic rolls). Registering the missing rank is a talent-adjacent change and was left.
5. Corruption: the client also lists 1223963 at level 60 with 137 per tick and 0.167; the engine registers 25311 (73 per tick, 0.2) as the earlier audit decided. If 1223963 is the live spell, Corruption is half of what it should be and the earlier audit's "halving" reading is wrong for it; worth a check against the beta.
6. Warlock pets (Imp's Firebolt, Succubus, Voidwalker, Felhunter) are not in `warlock.json` and keep their own tables; Rain of Fire's cast ids carry no damage effect in the client, so its row stays n/a in the report and is held by the table test only.
7. No talent file was touched; nothing here needs a `talents.go` edit.
