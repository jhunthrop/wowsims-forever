# Rogue: engine vs client spell conformance

## Spells

| Spec | Level | Spell | Rank | SpellID | Cost (client→engine) | CostType (client→engine) | Cooldown ms (client→engine) | CastTime ms (client→engine) | GCD ms (client→engine) | ReqLevel (client→engine) | Duration ms (client→engine) | Verdict | Diff | Damage min-max at level (client→engine) | Coefficient (client→engine) | Damage |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Rogue | 10 | Eviscerate | 1 | 2098 | 1.00→1.00 | mana→mana | 0→0 | 0→0 | 1500→1500 | 1→1 | n/a | match |  | n/a | n/a | n/a |

## Talent-gated spells

| Spec | Level | Spell | Rank | SpellID | Cost (client→engine) | CostType (client→engine) | Cooldown ms (client→engine) | CastTime ms (client→engine) | GCD ms (client→engine) | ReqLevel (client→engine) | Duration ms (client→engine) | Verdict | Diff | Damage min-max at level (client→engine) | Coefficient (client→engine) | Damage |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|

## Trainable abilities the engine does not register

Intro.

| Ability | Level (first→last) | Ranks | Skill line | Source | Cost | Cast ms | Cooldown ms | Why it matters |
|---|---|---|---|---|---|---|---|---|
| Pick Lock (1804) | 8→20 | 3 | n/a | class_spell | 90 energy | 0 | 0 | power cost |
| Distract (1725) | 8→20 | 3 | n/a | class_spell | 90 energy | 0 | 0 | power cost |
