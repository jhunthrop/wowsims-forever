# Mage: engine vs client spell conformance

Client build: x.

## Spells

| Spec | Level | Spell | Rank | SpellID | Cost (client→engine) | CostType (client→engine) | Cooldown ms (client→engine) | CastTime ms (client→engine) | GCD ms (client→engine) | ReqLevel (client→engine) | Duration ms (client→engine) | Verdict | Diff | Damage min-max at level (client→engine) | Coefficient (client→engine) | Damage |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Mage | 10 | Fireball | 1 | 133 | 1.00→1.00 | mana→mana | 0→0 | 0→0 | 1500→1500 | 1→1 | n/a | match |  | n/a | n/a | declared, matches |
| Mage | 10 | Frostbolt | 1 | 116 | 1.00→1.00 | mana→mana | 0→0 | 0→0 | 1500→1500 | 1→1 | n/a | match |  | n/a | n/a | n/a |
| Mage | 10 | Fire Blast | 1 | 2136 | 1.00→1.00 | mana→mana | 0→0 | 0→0 | 1500→1500 | 1→1 | n/a | mismatch |  | n/a | n/a | n/a |

## Talent-gated spells

| Spec | Level | Spell | Rank | SpellID | Cost (client→engine) | CostType (client→engine) | Cooldown ms (client→engine) | CastTime ms (client→engine) | GCD ms (client→engine) | ReqLevel (client→engine) | Duration ms (client→engine) | Verdict | Diff | Damage min-max at level (client→engine) | Coefficient (client→engine) | Damage |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Mage (Arcane Blast talent) | 20 | Arcane Blast | 1 | 400574 | 1.00→1.00 | mana→mana | 0→0 | 0→0 | 1500→1500 | 1→1 | n/a | client-scripted |  | n/a | n/a | n/a |

## Trainable abilities the engine does not register

Intro.

| Ability | Level (first→last) | Ranks | Skill line | Source | Cost | Cast ms | Cooldown ms | Why it matters | Disposition |
|---|---|---|---|---|---|---|---|---|---|
| Sleep (700) | 8→20 | 3 | n/a | class_spell | 90 mana | 0 | 0 | power cost | utility: crowd control |
| Conjure Water (5504) | 8→20 | 3 | n/a | class_spell | 90 mana | 0 | 0 | power cost | utility: creates a drink |
