# Damage conformance, melee lane (warrior, hunter, rogue)

Branch `damage-melee`, from `forever` at 7fd2a66f7. Client build 1.60.1.70009.

## Result

Level-60 damage block of `SUMMARY.md`:

| Class | before | after |
|---|---|---|
| Warrior | 0 declared, 12 not declared | 12 declared, 12 matches, 0 differs |
| Hunter | 0 declared, 11 not declared | 11 declared, 11 matches, 0 differs |
| Rogue | 0 declared, 6 not declared | 6 declared, 6 matches, 0 differs |

No `declared, differs` is left in these three classes. Every other
verdict column (match, mismatch, client-scripted) is unchanged in the three
goldens apart from the Sniper Shot and Summon Hawk rank rows below. The 18
`not declared` left in `SUMMARY.md` belong to Paladin (16) and Warlock (2).

No `.results` golden moved: the preset golden suites of all three
classes (`TestP1DPSWarrior`, `TestP1Hunter`, the rogue and tank warrior
suites) pass unchanged. The
changed numbers below are in spells those presets do not cast (or cast at
the same value). Full `go test --tags=with_db ./sim/...` passes except
`sim/web` (needs `binary_dist`, a pre-existing setup failure).

## How it is built

- Each class has `client_damage.go`: one `clientdamage.Effect` per rank,
  read from the generated `<Spell>BaseDamage/PointsPerLevel/Level/MaxLevel`
  through `clientdamage.FromTable` where the generated row is the effect the
  ability rolls for the ids it registers, and stated from the client file
  (with the reason in a comment) where it is not.
- Each class has `spellconst_damage_test.go`, written first, checking every
  rank at eight caster levels against `client/<class>.json` on the ids the
  engine registers (or, where the client keeps the damage in a separate
  spell, that spell's ids).
- Each ability sets `ClientBaseDamage` and rolls `Effect.Roll`. A flat
  effect draws nothing from the random stream, so flat spells are unchanged
  in the stream.
- `clientdamagetest/meleekinds.go` (new, shared test helper) adds the kinds
  `Dummy` (effect 3), `WeaponDamageNoSchool` (17), `WeaponDamage` (58) and
  `AreaPeriodic` (27, aura 3).
- Hand tables deleted: Revenge, Mortal Strike, Raptor Strike, Mongoose Bite,
  Counterattack, Wing Clip, Arcane Shot, Serpent Sting, Aimed Shot, the trap
  literals, Sinister Strike, Backstab, Ambush, Mutilate, Eviscerate's flat and
  width, Garrote, Rupture's base tick, Instant and Deadly Poison, and the
  rogue tests that pinned them.

## Values that disagreed with the client (fixed)

| Spell | Was | Client |
|---|---|---|
| Revenge | vanilla 12-14 .. 81-99 (rank 6) | 22/34/48/82/121/153, each about 20% wide (rank 6: 137.7-168.3) |
| Shield Slam | flat low end 640 (rank 4) | 655, 4.6% wide |
| Explosive Trap | one flat 115/163/229 | same centres, 26%/29%/24% wide, +0.8/1.0/1.2 per level to the rank's cap (level 60: 119.8, 169, 236.2 centres) |
| Sniper Shot | rank 1 (160) at every level, id 1310687 | ranks 160/225/295 at levels 40/48/58; the engine now follows the rank by level (ids 1310687/1310785/1310786) |
| Volley | 50/65/80 per tick (vanilla) | 70/91/112 (spells 1279721/1279719/1279715, the only Volley spells that carry a damage effect) |
| Summon Hawk | rank 1 at every level (32, cost 80) | ranks 32/47/85/108, cost 80/105/135/190 at levels 25/36/48/60 |
| Wing Clip | coefficient 0 | coefficient 1 (the report compared it) |
| Instant Poison | 19-25, 19-25, 44-56, 67-85, 67-85, 112-148 | 15, 23, 33, 51, 71, 88, each about a quarter wide |
| Deadly Poison tick | 13/13/20/20/34 | 9/13/20/27/34 |

Mortal Strike (85/110/135/160), Slam (16/32/43/68/87), Heroic Strike, Cleave,
Hamstring, Pummel, Thunder Clap, Rend, Execute, Overpower, Bloodthirst, Arcane
Shot, Serpent Sting, Aimed Shot, Raptor Strike, Mongoose Bite, Counterattack,
Immolation Trap (21/43/68/102/138 per tick), Sinister Strike, Backstab,
Ambush, Mutilate, Eviscerate (flat and width), Garrote and Rupture (base
tick) already agreed; they are now tables with tests. Whirlwind, Hemorrhage,
Ghostly Strike, Strider Kick, Multi-Shot, Venom, Wound Poison, Deep Wounds
and Lacerating Strikes have no flat, direct or periodic amount in the client
that the engine does not already model as a weapon percentage or a share of
the hit, so nothing is declared for them (Strider Kick, Lacerating Strikes'
amount of 1 and Deep Wounds' trigger are placeholders).

Generated-table faults worked around in `client_damage.go` (data lane): Mortal
Strike reads its healing aura, Slam's rows sit one tier off on stub ids, Aimed
Shot rank 6 reads the 600 duplicate (27632), Mutilate rank 1 reads the talent
spell, Revenge and Raptor Strike are skipped by the generator (hand
`Ranks` constants).

## Open questions for the owner

- Serpent Sting and Raptor Strike carry two id families in the client (the
  spellbook ids the engine registers, and a reissue at 425728+ / 4153xx with
  different amounts; Serpent Sting rank 9 agrees, Raptor Strike rank 8 is 70
  against the reissue's 140). The engine keeps the spellbook ids and their own
  amounts. The generated tables read the reissue.
- Volley and Instant Poison: the vanilla numbers stood because the registered
  ids carry no damage effect; the client states the damage on separate spells
  (see the table). Both move level-60 damage by tens of percent, so a
  second look at the tooltips is worth it before launch.
- Eviscerate and Rupture per-combo-point terms (EffectPointsPerResource) are
  not in the vendored client JSON; they stay typed in the ability files with
  their own tests.
- Explosive Trap's instant hit was described as "flat" from Wowhead's
  "Value: 116", which is centre plus one; the client file gives a width.

## Tests run

`go test --tags=with_db -count=1 ./sim/warrior/... ./sim/hunter/... ./sim/rogue/... ./sim/conformance/`
after `FOREVER_UPDATE_GOLDEN=1 go test --tags=with_db ./sim/conformance/`
(goldens: warrior, hunter, rogue, SUMMARY damage block).
