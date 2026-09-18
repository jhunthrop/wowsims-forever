# Forever fork: deliberate divergences from wowsims/classic

This fork tracks `upstream` = https://github.com/wowsims/classic. Everything
here is a change a merge must not silently revert.

## Committed generated protobufs

`sim/core/proto/*.pb.go` is committed; upstream ignores it. The Forever Sixty
site consumes this repository as a Go module at a pinned pseudo-version, and
the Go module system resolves packages from the commit, not from a build step.
`make proto` regenerates. `sim/core/proto/generated_test.go` fails if the
committed output drifts from `proto/*.proto`.

Requires: protoc >= 3.21 and protoc-gen-go v1.36.6
(`go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6`).

## wasm_exec.js location

Go 1.24 moved `wasm_exec.js` from `$GOROOT/misc/wasm` to `$GOROOT/lib/wasm`.
`vite.build-workers.ts` checks both.

## spell_mod.go

Ported from `wowsims/sod` `sim/core/spell_mod.go` at commit `0e3f6ef`
(https://github.com/wowsims/sod). Forever's reworked talents are mostly
"these spells cost, crit, or hit differently", which SoD expresses as
declarative `SpellModConfig` rather than as closures on
`OnSpellRegistered`. The straight copy needed eleven symbols classic
lacked; nine are plain additions (`SpellFlagNoSpellMods`,
`Spell.ClassSpellMask` / `SpellConfig.ClassSpellMask`, `Spell.Matches`,
`Spell.RelatedSelfBuff` / `SpellConfig.RelatedSelfBuff`, and the five
`Apply*DamageBonus` helpers), and two are deliberate divergences:

- **The five damage-bonus helpers** (`ApplyAdditiveBaseDamageBonus`,
  `ApplyMultiplicativeDamageBonus`, `ApplyAdditiveDamageBonus`,
  `ApplyAdditiveImpactDamageBonus`, `ApplyAdditivePeriodicDamageBonus`,
  in `sim/core/spell.go`) write classic's exported `float64` multiplier
  fields on `Spell` (`BaseDamageMultiplierAdditive`, `DamageMultiplier`,
  `DamageMultiplierAdditive`, `ImpactDamageMultiplierAdditive`,
  `PeriodicDamageMultiplierAdditive`) directly, instead of SoD's private
  `int64` percent accumulators recomputed into a cached multiplier in
  `updateImpactDamageMultiplier`. A SpellMod's percent (e.g. `-50`) is
  converted to a multiplier offset once, at call time.
- **`(*Cooldown).ApplyFlatCooldownMod` / `ApplyFlatPercentCooldownMod`**
  (`sim/core/cooldown.go`) are added directly to classic's `Cooldown`
  (a `*Timer` plus a `Duration`), mutating `Duration`, instead of classic
  adopting SoD's `SpellCooldown` wrapper that keeps a flat modifier and a
  percent multiplier separately and applies them lazily in a fixed order.
  No shipped talent applies both a flat and a percentage to the same
  cooldown, so applying each in call order is indistinguishable; if one
  ever does, this is the place to revisit.

No spell registers a `ClassSpellMask` yet (that starts with the specs that
use this system, Tasks 11 and 12), so no `.results` golden is affected by
this port.

## Build prerequisites for a fresh clone

    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
    export PATH=$PATH:$(go env GOPATH)/bin
    make proto
    make binary_dist/dist.go     # sim/web embeds this; it is gitignored
    go build ./...
    go test --tags=with_db ./sim/...
