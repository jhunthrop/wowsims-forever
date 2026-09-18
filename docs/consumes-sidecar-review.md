# Review: data/builds/<build>/simconsumes.json against sim/core/consumes.go

Reviewed: 2026-09-18. Build: 1.60.1.69893. Rows: not yet emitted.

`data/builds/1.60.1.69893/` has no `simconsumes.json` yet (checked
2026-09-18: the directory holds `classes.json`, `combos.json`,
`dungeons.json`, `icons/`, `items/`, `items.json`, `manifest.json`,
`races.json`, `sets.json`, `spells.json`, `talents/`, `talents.json` and
`zones.json`, and no `simconsumes.json`). `SimDatabase` carries no
consumables field, so this sidecar is the only path consumables take from
the pipeline into the engine, and this note answers the three questions
from `sim/core/consumes.go` (1,252 lines) alone, so the data lane has
something to build against before its first pipeline run.

## 1. Does every field the engine needs exist?

`sim/core/consumes.go` never applies a consumable by item id. It switches
on the `proto.Consumes` message's own enum fields and hard-codes each
enum value to a fixed effect:

- `consumes.Flask` (`proto.Flask_*`) → flat stats (`applyFlaskConsumes`,
  e.g. `FlaskOfSupremePower` → `+150 SpellPower`).
- `consumes.MainHandImbue` / `consumes.OffHandImbue`
  (`proto.WeaponImbue_*`) → flat stats or an on-hit proc aura, plus a
  shared 10s ICD for the two oils that proc (`addImbueStats`,
  `registerShadowOil`, `registerFrostOil`).
- `consumes.Food` (`proto.Food_*`) and `consumes.Alcohol`
  (`proto.Alcohol_*`) → flat stats; `consumes.DragonBreathChili` (bool) →
  a permanent 5%-proc aura (`applyFoodConsumes`).
- `consumes.DefaultPotion` (`proto.Potions_*`) → a `MajorCooldown` spell,
  keyed through `makePotionActivationInternal`, which is itself keyed by
  the enum value, not an item id (`registerPotionCD`).
- `consumes.DefaultConjured` (`proto.Conjured_*`) → a `MajorCooldown`,
  where the enum's *implementation* happens to hard-code an item id
  internally (e.g. `ConjuredHealthstone` → `makeHealthConsumableMCD(5509,
  ...)`) but the sidecar is never asked for that id — it is baked into
  the Go switch (`registerConjuredCD`).
- Defensive/physical/spell buff consumables, Zaza buffs, hit-rating
  consumables, and the `MiscConsumes` booleans (`BoglingRoot`,
  `RaptorPunch`, `JujuEmber`, `JujuChill`, `JujuFlurry`, `JujuEscape`) →
  flat stats or a registered aura/spell, each its own enum or bool field
  (`applyDefensiveBuffConsumes` through `applyMiscConsumes`).
- Explosives (`consumes.FillerExplosive`, `consumes.SapperExplosive`) →
  registered attack spells with fixed damage ranges
  (`registerExplosivesCD`).

So the field the engine needs per consumable is not "an item id" — it is
"which existing `proto.*` enum value (or `MiscConsumes` bool) this row
corresponds to, plus the numbers that enum's case already hard-codes."
Nothing in `consumes.go` reads a stat block from outside the binary
today, so there is no sidecar field the engine is currently missing to
keep behaving exactly as it does now. The open question is the reverse
one, in §3: which proto enum values (if any) have **no** case in
`consumes.go` and so are silently no-ops, and whether the sidecar should
be the thing that catches that drift instead of a manual read of this
file.

## 2. Does every row map onto something the engine can apply?

Unknown until the file exists — there are no rows to check yet. From the
code side, the ceiling on what the engine can apply today is:

- **Flat stats** (most flasks, all food/alcohol, most imbues, most
  defensive/physical/spell buffs, most Zaza buffs): straightforward,
  the engine already models `character.AddStats`/`AddStat` for every
  `stats.*` field these use.
- **Use effects with a `MajorCooldown`** (potions, conjured items,
  explosives): modeled, but each is a bespoke `SpellConfig` written by
  hand in this file — the sidecar cannot currently drive one of these
  generically, only confirm which enum value a build's item corresponds
  to.
- **Proc effects with an ICD** (Dragonbreath Chili, the weapon oils):
  modeled the same way, bespoke Go, not data-driven.

So today 100% of what `consumes.go` applies is a flat stat block or a
hand-written proc/use effect selected by enum value — there is no "the
engine can't apply this row" case yet, because nothing is read from data
at all. Once `simconsumes.json` exists, the real question becomes: how
many of its rows are new consumables (not yet a `proto.*` enum value or a
`case` in this file) versus reissues of ones already hard-coded here.
That split can't be answered until the file ships.

## 3. What the engine lane asks the data lane for

1. Keep `simconsumes.json` keyed by the same id space the client uses for
   the consumable item (`item_id`), plus the specific `spell_id` its use
   effect casts — `consumes.go`'s enum cases key off spell effects
   (`ActionID{SpellID: ...}`) more often than off the item id, and a join
   on spell id is what lets a beta log line confirm a proc or a cast.
2. For each row, the DB2 effect type (a flat-stat aura vs. a
   cast-on-use spell vs. a proc-on-hit aura) and its numbers (which
   stat(s) and amounts, or which spell it casts and that spell's own
   `spellconst` entry) — that is the one piece `consumes.go` cannot
   derive from a name alone, and is exactly the gap flat-stat consumables
   like `FlaskOfSupremePower` fill today with a literal this lane would
   otherwise have to keep re-typing by hand every patch.
3. A flag (or a distinguishable id range, matching the convention this
   lane already uses for `spellconst` — ids above 1,000,000 are new
   Forever content) marking which rows are new to Forever versus
   reissues of a vanilla consumable already hard-coded in `consumes.go`,
   so this lane can tell "add a new `case`" from "confirm a number
   against a literal already there" without diffing 1,252 lines by hand.
