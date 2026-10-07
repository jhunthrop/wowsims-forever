# Mage and priest damage: the abilities now roll the client's numbers

2026-10-07. Fork branch `damage-mage-priest`. Owner: move this file to `design/reviews/2026-10-07-damage-mage-priest.md`.

## Result

| Class | Declared | Matching | Differing | Not declared | n/a |
|---|---|---|---|---|---|
| Mage | 77 | 77 | 0 | 0 | 24 |
| Priest | 43 | 43 | 0 | 0 | 9 |

Before: mage 0 / 77 not declared, priest 0 / 43 not declared (rows are spec-and-rank pairs at level 60). Every `not declared` row became `declared, matches`; no `declared, differs` is left. Rows the report marks `n/a` stay `n/a`: Blizzard (the client's engine-id row has a dummy effect, its damage is on a sibling row), the Arcane Missiles channel and Devouring Plague (aura types the comparison does not read).

## How it is built

- `sim/common/clientdamage.Roll` turns an own-level `{min, max}` into the roll at the caster's level: centre grows by `PointsPerLevel` for each level above the spell's own, capped at `MaxLevel`, width kept. `clientdamagetest.AssertRoll` pins a registered roll to `spellconst.Spell.DamageRange` and is used by `sim/mage/spellconst_damage_test.go`, `sim/priest/spellconst_damage_test.go` and `sim/priest/shadow/client_damage_test.go` at caster levels 10/20/30/38/40/50/60 for every rank.
- Hand tables deleted where the generator's rank winner (higher spell level, then higher id) is the engine's id: the generator now emits `Ranks`, `SpellId`, `Level`, `CastTime`, `ManaCost`, `SpellCoeff`, `BaseDamage`, `PointsPerLevel`, `MaxLevel`. Abilities read those and set `ClientBaseDamage`.
- Hand tables kept, now client-valued and tested, where the generator would pick a different id and the engine's ids must not change (APL ids, logs): Fire Blast (generator picks 400616-400623), Arcane Blast (rank 3 would be 42896), Flamestrike (ground-aura ids), Blizzard (damage on sibling rows), Arcane Missiles channel (`ArcaneMissilesChannel*`: the channel's ids, cost, level, cast time; the generated `ArcaneMissiles*` arrays are the missile's), Shadow Word: Pain rank 8 (`sim/priest/client_damage.go`).
- Periodic effects the generator does not emit (Fireball, Pyroblast, Flamestrike, Holy Fire dots) carry a flat per-tick table, tested against the client's effect at every level; none grows with level. The coefficient of a pure-dot spell (Shadow Word: Pain, Mind Flay) is also set on the cast spell so the report reads it; the dot keeps its own.
- Talent modifiers are untouched.

## Spells changed (old hand table, client range at 60, coefficient)

| Spell (rank) | Old | Client at 60 | Coefficient |
|---|---|---|---|
| Scorch 7 | 237-280 | 166.4-196.4 | 0.429 |
| Fireball 12 direct | 596-760 | 424.6-541.4 | 1.0 (ranks 1-4: 0.123/0.271/0.5/0.793 became 0.429/0.571/0.714/0.857) |
| Fireball 12 dot | 76 total (19 x 4) | 15 x 4 = 60 | shared with the direct hit, as before |
| Frostbolt 11 | 515-555 | 457.2-492.8 | 0.814 |
| Fire Blast 7 | 446-524 | 415.4-490.6 | 0.429 (ranks 1-2 were 0.204/0.332) |
| Arcane Missiles 8 | 230 per missile | 209 per missile (trigger 25346) | 0.286 (was 0.24) |
| Pyroblast 8 | 716-890, dot 268 total | 519.8-646.2, dot 53 x 4 = 212 | 1.0, dot 0.15 |
| Arcane Blast 5 | flat 394 | 364.2-423.8 | 0.714 |
| Arcane Explosion 6 | 249-270 | 238.7-258.3 | 0.143 |
| Blast Wave 5 | 462-544 | 452.8-533.2 | 0.129 |
| Flamestrike 6 | 381-466, dot 340 total | 381.1-466.5, dot 83 x 4 | 0.157, dot 0.032 (was 0.02) |
| Blizzard 6 | 1192 total (149 x 8) | 146 x 8 | 0.042 |
| Frost Nova 4, Ice Lance 5-6 | own-level pair, no growth | grow with level | table / convention |
| Mind Blast 9 | 508-537 | 476.9-503.5 | 0.429 |
| Shadow Word: Death 4 | flat 448 | 444.1-471.9 | 0.429 |
| Shadow Word: Pain 8 | 852 total (142 x 6) | 127 per tick | 0.2 |
| Mind Flay 6 | 426 total | 130 per tick | 0.167 (was hand 0.15) |
| Devouring Plague 6 | 904 total | 106 per tick x 8 | 0.1 |
| Smite 8 | 384-429 | 166.6-186.4 | 0.714 |
| Holy Fire 8 | direct 355-449, dot 145 total | 183.7-232.3, dot 15 x 5 | 0.75, dot 0.05 |
| Starshards 7 | 936 total (156 x 6) | 300 per tick | 0.167 |

Mind Flay and Starshards also lost a second division of the tick amount inside `ExpectedTickDamage`, so their expected-tick figures rise.

## Goldens moved

- `sim/mage/TestP1Mage.results` (Frost, the only mage golden): Phase1-Average-Default DPS 657.38 to 620.25 (-5.6%), TPS 605.96 to 572.55; stat-weight intellect 0.163 to 0.153, hit 7.157 to 6.750, crit 4.240 to 4.000. 77 entries moved, all down; the cause is Frostbolt 515-555 becoming 457-493 (and the same mean on the other ranks). Adopted.
- `sim/core/testdata/conformance/{mage,priest}.golden.md` and the damage block of `SUMMARY.md` regenerated: only the Damage columns moved.
- `TestP1Shadow` skips here (no item database), so the priest DPS movement is unmeasured; the nightly will show it.

## Open

1. Fire Blast, Flamestrike and Blizzard: the client carries two row families for each (2136-10199 and 400616-400623; 2120-10216 and 1279976-1279990). The engine casts the old ids (the APL uses 10199), so the numbers follow those rows; the 400xxx Fire Blast rows say 470 at rank 7 and coefficients 0.204/0.332 at ranks 1-2. If the live client casts the 400xxx ids, the ids and the table need one decision.
2. The conformance report cannot see Arcane Missiles damage: the channel row is an aura and the missile is a passive spell the report skips. The missile declares `ClientBaseDamage` and the test pins it to 25346; the row stays `n/a`.
3. The generator emits only the primary effect per rank, hence the small hand tables for dots and the id overrides above; secondary-effect arrays and a pick-the-engine's-id rule would remove them.
4. Fireball's dot still shares the direct hit's coefficient (the client states 0 for it). Whether the dot should take spell power at all is a model question.
5. `TestCastingOnlyMovementIsFreeForTheFuryWarrior` in `./sim` fails in this worktree; it is a warrior test and nothing here touches it.
