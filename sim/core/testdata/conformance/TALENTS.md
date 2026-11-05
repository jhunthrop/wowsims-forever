# Talent rates against Forever's talent text

Hand-written, like SUMMARY.md. The conformance report (`sim/conformance`) reads
spell rows, not talent rank values, so a talent whose per-rank rate the engine
carried over from vanilla was never checked. This file is that check, for every
class, against the site's `data/builds/1.60.1.70532/talents/<class>.json`
(mage, druid and paladin: `1.60.1.70707`)
(trees, talents, `max_rank`, and a `description` per rank: the live,
hotfix-aware Forever text).

For each talent the engine reads (a field of `<Class>Talents` that something
under `sim/<class>/`, `sim/core` or `sim/common` reads), every number in the
rank-1 and max-rank text was compared with what the engine applies at that
rank: the constant, the per-rank multiplier, the duration, the table entry.
The rank-by-rank text was also compared where the engine uses a table, since
Forever's rates are not always linear (Barrage 3/7/10, Improved Scorch
33/67/100, Winter's Chill stacks).

Verdicts:

- **match**: every number the engine applies equals the text. A note names any
  clause of the text the engine does not apply (a stun chance, a range, a
  threat cut); those clauses are unmodeled halves, not mismatches.
- **mismatch, fixed**: the engine's number differed; the engine now uses the
  Forever value, with a unit test that pins the rate.
- **unmodeled**: the engine reads the field only to keep it from being
  reported as unused. Nothing changes in a simulation. Several of these are
  real DPS effects the engine could model (Holy Precision, Power in Light,
  Spiritual Guidance's spell-damage half); they are listed in the report as
  follow-ups.
- **text-only**: the talent is modelled but its number cannot be read off the
  text (a template that did not fill, or a spell whose numbers the spell-row
  conformance report already checks); the number the engine uses is given.
- **unsettled**: the text has two readings, or the engine follows a source the
  text does not; nothing was changed. Both readings are listed.
- **not read**: no engine field, or nothing reads it.

Rates are per rank; "Engine at max rank" is the value the engine applies at the
talent's maximum rank unless a note says otherwise.


## Counts

| Class | Checked | Match | Mismatch (fixed) | Unmodeled | Text-only | Unsettled | Not read |
|---|---|---|---|---|---|---|---|
| Druid | 52 | 40 | 5 | 6 | 0 | 1 | 0 |
| Hunter | 50 | 30 | 4 | 15 | 1 | 0 | 0 |
| Mage | 52 | 36 | 4 | 12 | 0 | 0 | 0 |
| Paladin | 51 | 26 | 15 | 10 | 0 | 0 | 0 |
| Priest | 53 | 31 | 5 | 15 | 2 | 0 | 0 |
| Rogue | 53 | 40 | 2 | 11 | 0 | 0 | 0 |
| Shaman | 50 | 33 | 6 | 8 | 3 | 0 | 0 |
| Warlock | 52 | 31 | 11 | 10 | 0 | 0 | 0 |
| Warrior | 47 | 41 | 1 | 4 | 0 | 1 | 5 |

"Checked" counts talents whose engine field is read somewhere under `sim/<class>/`, `sim/core` or `sim/common`; a field only touched by a `_ =` line counts as checked and is an "unmodeled" row. "Not read" talents have no engine field or nothing reads it. Hunter also had Ferocity and Unleashed Fury fixed earlier, in `ef964a5aa`; they are listed as matches.

## Unsettled

- **Druid: Thick Hide.** Engine: 1 armor per level per rank, 2/3 per defense point per rank. Reading A: "180 additional base Armor per level" is the same at every rank (the defense term alone grows 0.67/1.33/2.00), so rank 1 gives 180 a level. Reading B: the 180 is the max-rank total shown as a constant (3 a level at level 60), as the engine has it. The defense term matches either way. The client rows do not settle it either: the aura is type 4 with base 100, and a second effect has base 3 with 3 per level, neither of which says whether the value is per rank or the max-rank total. Left as is.
- **Warrior: Improved Slam.** Engine: cast and GCD -0.5 sec; cooldown -3 sec a rank = -6 sec at 2/2. Reading A (the 70291 text): "Slam's cooldown is reduced by 3.0 sec" at both ranks, 3 sec flat; Reading B (the engine, from Blizzard's 1 October 2026 notes "3 s off the cooldown per rank"): 6 sec at 2/2. The client rows do not tell the readings apart: spell 12862 is one spell for both ranks, with base -3000 ms on the cooldown and -500 ms on the cast time, so a per-rank reading cannot be told from a flat one. The engine is left as is.

## Client 1.60.1.70425 (2026-10-16)

The tree layouts moved (Arcane gained Arcane Wrath, Shadow lost its tier-2 Improved Mind Blast node, Elemental and Restoration reshuffled, three talents were renamed in place), and these rates and mechanics changed with them. Each row above carries the new text.

- **Mage: Arcane Blast.** Stacks grant 15% of Intellect as spell power (was +10% other-spell damage); 15 sec from the first stack, not renewed and not dropped by another spell; rows 32/78/101/168/236, coefficient 0.429.
- **Mage: Arcane Wrath.** New talent nuke: 3.5 sec cast, 6 sec cooldown, +5% of maximum mana as spell damage; feeds Missile Barrage.
- **Mage: Missile Barrage.** Two ranks: 20/5/10% a rank on Arcane Blast / Arcane Missile hits / the bolts.
- **Mage: Fire Power.** 1% a rank per Fire effect on the target, four at most (was a flat 2% a rank).
- **Mage: Critical Mass.** Five ranks at 2% (was three).
- **Mage: Heating Up.** A landed Fire Blast adds a stack, a critical one two.
- **Mage: Ice Lance.** x3 on a Frozen target (was x4).
- **Mage: Winter's Chill.** Five stacks at every rank; +40% Cone of Cold damage a stack; the 2% crit a stack is gone.
- **Priest: Mental Strength.** 2% Intellect a rank (was 3%).
- **Priest: Improved Mind Flay.** Each tick is 10% a rank more than the one before (was a flat +10% a rank).
- **Priest: Dark Inspiration.** New: +3% Spirit a rank for 10 sec from Mind Blast or a full Mind Flay.
- **Priest: Divine Aegis.** Capped at 20 a level.
- **Priest: Shadow Affinity, Improved Fade.** Gone from the tree.
- **Warlock: Shadow Mastery.** 2% a rank (was 1%). Malediction became Grim Reach (range).
- **Shaman: Call of Flame.** Flame Shock +3 sec a rank. Trial by Fire replaces Improved Fire Nova and adds Magma Totem +10 sec a rank. Call of Thunder is gone. Water Shield spends a globe on any spell crit.
- **Paladin: Champion of the Light.** 10/20/30% (was 20/40/60%). Shield Specialization restores 4% mana (was 6%).
- **Hunter: Resourcefulness.** 50/100% regen chance (was 30/60%).
- **Druid: Natural Instinct.** Predatory Instincts' slot, plus healing from 12/25% of Intellect.

## Client 1.60.1.70532 (2026-10-24)

Frost was reworked (Piercing Ice is gone and its node order moved; the tree is 18 nodes) and every "avoid interruption" talent was reworded as a chance to not lose casting time. Each row above carries the new text.

- **Mage: Piercing Ice.** Gone from the tree; the engine's 2% a rank Frost damage went with it. The reference build (`ForeverFrostTalents`) carries its three points into Elemental Precision.
- **Mage: Elemental Precision.** Three ranks of 2% (was five of 1%).
- **Mage: Winter's Chill.** Frostbolt, Frost Blast and Fingers of Frost empowered Ice Lance apply it (was every Frost damage spell); the mage's next Cone of Cold takes +5% damage a stack and +7% of spell damage a stack and spends the stacks (was +40% a stack, not spent). Frost Blast is not registered.
- **Mage: Ice Barrier.** +3% Frost damage for 1 min from the cast and +3% more while the shield holds; the absorb now scales with Frost spell power. The absorb stays unmodeled.
- **Mage: Missile Barrage.** Arcane Missile hits no longer proc it; Frost Blast joins the 10% a rank bolts.
- **Mage: Frost Warding.** Frost Ward negates instead of reflecting, and Ice Barrier and Frost Ward absorb 10% a rank more. Still unmodeled.
- **Paladin: Champion of the Light.** 20/40/60% (was 10/20/30%).
- **Warlock: Amplify Curse.** Curse of Exhaustion +20% (was +50%); the row already read 20%.
- **Pushback talents.** Nature's Focus, Improved Channeling, Twilight Focus, Healing Focus, Eye of the Storm, Intensity and Fel Concentration are "a N% chance to not lose casting time" rather than a chance to avoid interruption or a pushback reduction. Pushback is not rolled in the fights modeled, so no engine number moves.

## Client 1.60.1.70685 and 1.60.1.70707 (2026-10-30)

Arcane lost a node (18) and Fire moved one; 70707's mage trees are 70685's. Each row above carries the new text.

- **Mage: Arcane Focus.** Gone from the tree; the engine's 1% a rank Arcane hit went with it. The reference build (`ForeverFrostTalents`) moves its five points to Improved Channeling, so the top row still opens the tree.
- **Mage: Precision.** Elemental Precision renamed; 2% a rank hit with every mage spell (was Frost and Fire).
- **Mage: Accelerant.** Improved Fireball renamed; Arcane Wrath joins Fireball and Frostfire Bolt.
- **Mage: Critical Mass.** Three ranks at 2% (was five); Combustion's prerequisite is 3/3.
- **Mage: Incineration.** Five ranks at 2% (was three).
- **Mage: Improved Fire Blast, Missile Barrage, Arcane Resilience.** Moved (Improved Fire Blast to Fire's third row; Missile Barrage and Arcane Resilience swapped). No rate changed.
- **Mage: Ice Lance.** Rank 1 (spell 1312002, levels 20 to 27) has an 8 sec cooldown in the client's spell rows; the trained ranks have none (ice_lance.go).
- **Hunter: Trueshot Aura.** The talent's text reads 15 ranged attack power at rank 1 (was 30); the engine reads the trained ranks' amounts (`TrueshotAuraRanks`), not the talent text.
- **Druid: the 70707 hotfix rework.** Berserk, Improved Shifting Power and Gift of the Earthmother are gone (the Grovekeeper Rage 5-piece, which shortened Berserk, is now NoSim). Heart of the Wild moved to Restoration and Furor to Feral Combat, rates unchanged. Several of the new numbers are in no client row (the tables are pre-hotfix); they come from Wowhead's live Forever tooltips, read 2026-10-31.
- **Druid: Primal Bite.** The Feral capstone (tier 6, learned at 40); 42 at 40, 62 at 50, 77 at 60 (client rows 26/38/59/77 at 25/36/48/60).
- **Druid: Shifting Power.** Tier 6 behind Predatory Discipline; 8 sec cooldown (client rows 16 sec).
- **Druid: Lacerate.** A talent (tier 3, behind Savage Fury 2); two new low ranks (20 over 15 sec at 25, 30 at 34); the hit is 10% weapon damage per existing stack (was a flat 20%).
- **Druid: Predatory Discipline.** New: Ferocious Bite keeps its excess-energy damage but no longer spends the energy.
- **Druid: Bloom.** New: a Rejuvenation or Regrowth that runs its duration heals for 100% of Spirit.
- **Druid: Genesis, Naturalist.** 2% a rank (were 1%); Naturalist's cast-time cut also reaches Wild Growth, which now has a 3 sec cast.
- **Druid: Moonkin Form, Leader of the Pack.** Their 3% now reaches all crit; the engine's unified Crit stat already applied it so.

### Paladin (1.60.1.70707)

The paladin trees were reworked: Twist of Light, Instrument of Law, One-Handed Weapon Specialization, Voice of Truth and Improved Seal of Fury are gone (their engine code with them); Divine Intellect moved to Protection, Holy Conduit to Holy and Unyielding Faith to Retribution; Oath of Light, Boundless Light, Shieldsworn, Stoicism, Turn Evil and Unsealed Wrath are new. Each row above carries the new text; the rate changes are the "mismatch, fixed" rows, each pinned by a unit test in `sim/paladin`. The reference Protection build (`ForeverProtectionTalents`) was re-carried by name onto the new tree.

## Client 1.60.1.70707 hotfixes (2026-10-31)

70707's talent text (the Wowhead overlay) carries number changes the shipped SpellEffect rows do not; the text is the source.

- **Mage: Ice Barrier.** +15% Frost damage for 1 min and +5% more while shielded (was 3% + 3%).
- **Mage: Winter's Chill.** Each stack is +20% damage taken from the next Cone of Cold (was +5%), still +7% of spell damage a stack, 5 stacks.
- **Mage: Ice Lance.** +300% against a Frozen target, x4 (was +200%, x3).
- **Mage: Arcane Blast.** 12% of Intellect as spell damage a stack (was 15%), cost +100% a stack (was +175%), 18 sec from the first cast (was 15).
- **Mage: Arcane Resilience.** Three ranks, 75% of Intellect as armor. Still unmodeled.
- **Priest: Binding Heal.** Gone with its talent; Holy Binding (Holy tier 3, a self-heal on Flash Heal) takes the slot, unmodeled. Renewed Hope's list drops Binding Heal. The shipped SkillLineAbility rows still list Binding Heal's ranks, so the conformance report shows it as a trainable the engine does not register.
- **Hunter: Aspect of the Hawk.** Ranks 1-5 are 15/22/30/38/47 ranged attack power (was 20/35/50/70/90); rank 6 (55) is now the strongest a level-58 hunter has, so the engine casts it.
- **Hunter: Trueshot Aura.** Trained ranks 15/22/30/40/50 (was 30/40/50/75/50); rank 5 is no longer inferior.
- **Hunter: Efficiency.** The class mask gains Strider Kick.
- **Hunter: Pathfinding, Improved Revive Pet.** Moved in the Beast Mastery tree. No rate changed.
- **Warlock: Improved Drains.** Drain Mana replaces Wrack in the list; Wrack no longer gains the 20%.
- **Orc: Blood Fury.** 2 min cooldown in the client's SpellCooldowns (20572); the engine already had 2 min.

## Mismatches fixed in this lane

- **Hunter: Efficiency.** Now 15% (was 10%). 3% a rank; was 2% a rank on Shots, Stings and Volley.
- **Hunter: Barrage.** Now 10% on Multi-Shot, Aimed Shot, Volley (3/7/10%; was 15% on Multi-Shot only, plus a Volley crit-bonus term). Table 3/7/10, not linear.
- **Hunter: Savage Strikes.** Now 4% (was 20%). 2% a rank on Raptor Strike and Mongoose Bite; was 10% a rank. Other melee abilities (Wing Clip, Counterattack, Strider Kick) still carry no bonus.
- **Mage: Improved Scorch.** Now 33/67/100% chance, 3% a stack to 5 stacks, 30 sec (was 33/66/100%). Rank 2 was 66%.
- **Mage: Winter's Chill.** (70291) 20% a rank chance, 2% crit a stack, stacks to the rank. Superseded by 1.60.1.70425 below: five stacks at every rank, +40% Cone of Cold a stack, no crit.
- **Paladin: Vindication.** Now AP +3% for 30 sec (was +15%). Was 5% a rank; the target Attack Power cut in the text is a garbled template ("(3 /- 3 *  - 204)") and is not modeled; the proc chance is not in the text; the aura row states ProcChance 100, so the engine proc on every landed melee hit matches.
- **Paladin: Vengeance.** Now 3% a stack, 3 stacks, 30 sec = 9% (was 9% flat for 8 sec, no stacks, at 3/3 from a 3%-a-rank table). 1% a rank a stack at 1/2/3: 1/2/3% after one crit; the old table gave 3% a rank on one application.
- **Priest: Silent Resolve.** Now 30% less Holy threat (was 12% less of all threat). Vanilla 4% a rank on all spells; stun/fear/silence duration half not modeled.
- **Priest: Searing Light.** Now Holy damage +2/5% (was 5/10%). Smite and Holy Fire only; the Holy Nova free-cast chance is not modeled.
- **Priest: Shadow Focus.** Now 5% Shadow hit (was 10% on every priest spell). 2% a rank on every priest spell.
- **Priest: Shadow Affinity.** (70291) 30% less Shadow threat. The talent left the tree in 1.60.1.70425.
- **Priest: Shadowform.** Now Shadow +10%, Shadow cost -50%, Shadow crit damage bonus +100%, Physical damage taken -15% (was +15% Shadow damage only).
- **Rogue: Deflection.** Now 6% parry (was 3%). 1% a rank before.
- **Rogue: Elusiveness.** Now Vanish cooldown -90 sec; Evasion no longer cut (rank-2 Evasion cooldown was 5 sec - 90 sec, negative). Blind not registered. Evasion is only reachable through RegisterEvasionSpell, which nothing calls.
- **Shaman: Improved Fire Nova.** (70291) +20% damage, -4 sec cooldown. Trial by Fire took its slot in 1.60.1.70425 with the same two halves plus Magma Totem duration.
- **Shaman: Ancestral Knowledge.** Now +10% Intellect (was +5% maximum Mana). 1% max Mana a rank before.
- **Shaman: Elemental Weapons.** Now Rockbiter +20%, Windfury +40%, Flametongue/Frostbrand +15%. Rockbiter rank 2 was 14% (text 13%).
- **Shaman: Anticipation.** Now 6% dodge (was 3%). 1% a rank before.
- **Shaman: Toughness.** Now +10% Stamina (was +10% armor from items). Target stat was wrong.
- **Shaman: Flurry.** Now +25% attack speed (was +30%); +5% at 1/5 (was +10%). Vanilla 10/15/20/25/30.
- **Warlock: Improved Corruption.** Now -2 sec cast, +10% damage (was cast time only). Damage half was unread.
- **Warlock: Shadow Mastery.** Now +5% Shadow damage (was +10%). 2% a rank before. Scope fixed from the client rows: the two effects of spell 18271 carry class masks (524435,0,0) and (17418,262147,0), which cover Shadow Bolt, Corruption, Curse of Agony, Death Coil, Drain Life, Drain Soul and Siphon Life. The engine withheld the multiplier from four of them (vanilla's exclusions), modded base damage on three, missed Drain Soul and double-applied Siphon Life; all seven now take +1% a rank as one multiplier, and spells outside the masks (Shadowburn, Curse of Doom, Wrack) no longer do.
- **Warlock: Demonic Embrace.** Now +15% Stamina, no Spirit change (was -5% Spirit). Vanilla Spirit penalty removed.
- **Warlock: Unholy Power.** Now +10% pet damage (was +20% on pet melee only). 2% a rank, pet-wide: every demon damage source, Firebolt included (round 3).
- **Warlock: Demonic Sacrifice.** Now Imp +15% Shadow, Succubus +15% Fire, Voidwalker 2% Mana / 4 sec, Felhunter 3% Health / 4 sec. The four effects were swapped (Imp Fire, Succubus Shadow, Voidwalker Health, Felhunter Mana).
- **Warlock: Soul Link.** Now caster takes 70% (was 76.9%), +3% damage both. Was /1.3.
- **Warlock: Master Demonologist.** Now Imp +10% Fire, Voidwalker -10% Physical taken, Succubus +10% Shadow, Felhunter -10% Magic taken. Was Imp -20% threat, VW -10% all damage taken, Succubus +10% all damage, Felhunter +10 resistance.
- **Warlock: Bane.** Now -0.5 sec Shadow Bolt, Immolate, Incinerate; -2 sec Soul Fire. Incinerate was not cut.
- **Warlock: Ruin.** Now +100% Destruction crit damage bonus, 20% a rank (was +100% at any rank). Below 5/5 it overstated.
- **Warrior: Enrage.** Now 30% chance, +10% Physical damage, 12 sec, on any damaging hit taken (was 100% on a melee crit taken, 12 attack charges). The chance was an unresolved template, the engine kept vanilla's trigger.
- **Mage: Ice Lance.** Now x4 against Frozen targets (was x3). SpellEffect 1342606 (spell 1312002, effect 3) has base points 300, i.e. +300% damage.
- **Warlock: Improved Shadow Bolt.** Now no charge limit, 12 sec (was 4 charges). Spell 17794 has SpellDuration 12000 ms and no SpellAuraOptions row.

## Druid

| Talent | Tree | Max | Forever text at max rank | Engine at max rank | Verdict | Note |
|---|---|---|---|---|---|---|
| Improved Wrath | Balance | 5 | Reduces the cast time of your Wrath spell by 0.5 sec and its Mana cost by 50%. | same | match |  |
| Genesis | Balance | 5 | Increases the periodic damage and healing done by your spells and abilities by 10%. | 2% a rank (10% at 5/5) | mismatch, fixed | client 1.60.1.70707 hotfix doubled it; the client row still says 1 a rank |
| Moonglow | Balance | 3 | Reduces the Mana cost of your damaging spells by 25%. | same | match |  |
| Improved Moonfire | Balance | 2 | Increases the damage and critical strike chance of your Moonfire spell by 10%. | same | match |  |
| Nature's Majesty | Balance | 2 | Increases your critical strike chance with spells and melee attacks by 4%. | same | match |  |
| Nature's Reach | Balance | 2 | Increases the range of your offensive Balance spells by 20% and improves your chance to hit by 4%. | 4% hit | match | range half not modeled |
| Improved Entangling Roots | Balance | 3 | Increases the damage done by your Entangling Roots spell by 75%, and its victims can take up to 75% more damage without interrupting the effect. | - | unmodeled | Entangling Roots not registered |
| Nature's Splendor | Balance | 1 | Increases the duration of your Moonfire and Rejuvenation spells by 3 sec, your Regrowth spell by 6 sec, and your Insect Swarm spell by 2 sec. | same | match |  |
| Insect Swarm | Balance | 1 | The enemy target is swarmed by insects, decreasing their chance to hit with attacks by 2% and causing 48 Nature damage over 12 sec. | same | match |  |
| Vengeance | Balance | 5 | Increases the critical strike damage bonus of your Arcane and Nature spells by 100%. | same | match |  |
| Improved Starfire | Balance | 5 | Reduces the cast time of Starfire by 0.5 sec and Starfire has a 15% chance to stun its target for 3 sec. | -0.5 sec cast | match | the 3-15% stun chance is not modeled |
| Overgrowth | Balance | 2 | Increases the maximum number of targets you may have affected by Entangling Roots by 2. | - | unmodeled | Entangling Roots not registered |
| Nature's Grace | Balance | 1 | All non-periodic spell criticals grace you with a blessing of nature, increasing your spellcasting speed and reducing your global cooldown by 10% for 3 sec. | same | match |  |
| Eclipse | Balance | 3 | Your Wrath spell reduces the cast time of your next 2 Starfire spells by 0.50 sec. Stores up to 4 charges. Lasts 15 sec. | same | match |  |
| Moonfury | Balance | 5 | Increases the damage done by your Arcane and Nature spells by 10%. | same | match |  |
| Moonkin Form | Balance | 1 | Shapeshift into Moonkin Form, increasing Omen of Clarity's chance to trigger by 100%, Armor contribution from items by 360%, and all party or raid members within 45 ya... | 3% Crit to the raid | match | the aura now reaches all crit; the engine's unified Crit stat already did. Armor/Omen halves not modeled |
| Ferocity | Feral Combat | 5 | Reduces the cost of your Maul, Primal Bite, Swipe, Claw, and Rake abilities by 5 Rage or Energy. | same | match |  |
| Furor | Feral Combat | 5 | Gives you a 100% chance to gain 10 Rage when shapeshifting into Bear Form or Dire Bear Form. Shapeshifting into Cat Form restores 100% of the Energy you had when last ... | same | match | Feral Combat tier 0 since 1.60.1.70707 (Restoration before) |
| Feral Swiftness | Feral Combat | 2 | Increases your movement speed while in Cat Form by 30%, and increases your chance to Dodge by 4%. | 4% dodge | match | cat movement speed not modeled |
| Feral Instinct | Feral Combat | 3 | Increases damage done by your Swipe ability by 30% and reduces the chance enemies have to detect you while Prowling as if you were 3 levels higher. | 30% Swipe damage | match | prowl detection not modeled |
| Brutal Impact | Feral Combat | 2 | Increases the stun duration of your Bash and Pounce abilities by 1 sec and reduces the cooldown of Bash by 30 sec. | - | unmodeled | Bash and Pounce stun duration |
| Thick Hide | Feral Combat | 3 | While in Bear Form, Cat Form, Dire Bear Form, or Moonkin Form, you gain 180 additional base Armor per level and another 2.00 base Armor for each point of defense skill... | 1 armor per level per rank, 2/3 per defense point per rank | unsettled | Reading A: "180 additional base Armor per level" is the same at every rank (the defense term alone grows 0.67/1.33/2.00), so rank 1 gives 180 a level. Reading B: the 180 is the max-rank total shown as a constant (3 a level at level 60), as the engine has it. The defense term matches either way. The client rows (aura 4 base 100; base 3 with 3 per level) do not resolve the per-rank question |
| Shredding Attacks | Feral Combat | 3 | Reduces the Energy cost of your Shred ability by 18 and reduces the Rage cost of your Lacerate ability by 3. | same | match |  |
| Savage Fury | Feral Combat | 2 | Increases the damage caused by your Claw, Rake, Shred, Maul, and Swipe abilities by 10%. | same | match |  |
| Feral Charge | Feral Combat | 1 | Requires Bear Form, Dire Bear Form Charge an enemy, immobilizing them and interrupting any spell they are casting for 4 sec. | - | unmodeled | gap closer |
| Sharpened Claws | Feral Combat | 2 | Increases your critical strike chance while in Bear Form, Dire Bear Form, or Cat Form by 6%. | same | match |  |
| Predatory Strikes | Feral Combat | 3 | Increases your melee Attack Power in Cat Form, Bear Form, and Dire Bear Form by 150% of your level. | same | match |  |
| Lacerate | Feral Combat | 1 | Lacerates the enemy target, making them bleed for 20 damage over 15 sec plus 10% weapon damage per existing application of Lacerate on the target. Causes a high amount... | 20 over 15 sec at 25 (1328617), 30 at 34, then the trained 50/60/75; hit 10% weapon damage per existing stack | mismatch, fixed | a talent since 1.60.1.70707 (trained at 42 before); the hit was a flat 20% weapon damage; the two low ranks are live tooltips, not client rows |
| Predatory Discipline | Feral Combat | 1 | Your Ferocious Bite ability no longer converts extra Energy into damage, but still deals extra damage based on your excess Energy. | Ferocious Bite keeps its excess-energy damage and no longer spends the energy | match | new in 1.60.1.70707 |
| Blood Frenzy | Feral Combat | 2 | Gives you a 100% chance to gain an additional 5 Rage any time you get a critical strike while in Bear Form or Dire Bear Form. In addition, your non-periodic critical s... | same | match |  |
| Leader of the Pack | Feral Combat | 1 | While in Cat Form, Bear Form, or Dire Bear Form, the Leader of the Pack increases the Critical Strike chance of all party or raid members within 45 yards by 3%. | 3% Crit to the raid | match | the aura now reaches all crit; the engine's unified Crit stat already did |
| Natural Instinct | Feral Combat | 2 | Increases the critical strike damage bonus of your melee abilities by 20% and increases your spell healing by 25% of your Intellect. | 10% a rank melee crit damage; healing 12/25% of Intellect as a stat dependency | match | Predatory Instincts' slot since 1.60.1.70425 |
| Rend and Tear | Feral Combat | 5 | Increases damage done by your melee abilities on Bleeding targets by 10%. | same | match |  |
| Natural Reaction | Feral Combat | 5 | Increases your dodge chance by 5%, and gives you a 100% chance to gain 5 Rage each time you dodge. | same | match |  |
| Primal Bite | Feral Combat | 1 | Bite the target, dealing 100% normal damage plus 42 and generating a high amount of threat. | 42 at 40, 62 at 50, 77 at 60 | mismatch, fixed | capstone since 1.60.1.70707; live tooltips raise the low ranks (client rows 26/38/59/77) |
| Shifting Power | Feral Combat | 1 | Instantly convert 0 Mana into 40 Energy. Shifting Power's cost is reduced by effects that reduce the cost of Shapeshifting. | 40 energy, 8 sec cooldown | mismatch, fixed | tier 6 since 1.60.1.70707; Improved Shifting Power is gone and the live tooltip states 8 sec (client rows 16 sec) |
| Nature's Focus | Restoration | 5 | Gives your Arcane and Nature spells a 70% chance to not lose casting time when you take damage. | - | unmodeled | pushback avoidance never triggers in the fights modeled |
| Heart of the Wild | Restoration | 5 | Increases your Intellect by 10%. In addition, while in Bear Form or Dire Bear Form your Stamina is increased by 20% and while in Cat Form your Strength is increased by... | same | match | Restoration tier 0 since 1.60.1.70707 (Feral before) |
| Naturalist | Restoration | 5 | Reduces the cast time of your Healing Touch and Wild Growth spells by 0.5 sec and increases all damage you deal by 10%. | 2% a rank all damage; 0.1 sec a rank off Healing Touch and Wild Growth | mismatch, fixed | 1.60.1.70707 doubled the damage and added Wild Growth, which now has a 3 sec cast (live tooltips) |
| Reflection | Restoration | 3 | Allows 50% of your Mana regeneration to continue while casting. | 50% (rank/3 of 50%: 16.7/33.3/50) | match | the client row is base 50 at 3/3, so the engine's exact thirds are right and the text's 17% is display rounding |
| Natural Shapeshifter | Restoration | 3 | Reduces the mana cost of all shapeshifting by 30%. | same | match |  |
| Subtlety | Restoration | 3 | Reduces the threat generated by your Nature and Arcane spells by 30%. | - | unmodeled | threat only |
| Gift of Nature | Restoration | 5 | Increases the effect of all your healing spells by 10%. | same | match |  |
| Bloom | Restoration | 1 | When your Rejuvenation or Regrowth spell expires due to its duration, the target is instantly healed for an amount equal to 100% of your Spirit. If either spell expire... | heal for 100% of Spirit when Rejuvenation or Regrowth runs its duration | match | new in 1.60.1.70707; the dispelled half never triggers (nothing dispels); crit and healing modifiers unconfirmed |
| Tranquil Spirit | Restoration | 5 | Reduces the mana cost of your Healing Touch and Tranquility spells by 10%. | same | match |  |
| Improved Rejuvenation | Restoration | 3 | Increases the effect of your Rejuvenation spell by 15%. | same | match |  |
| Swiftmend | Restoration | 1 | Instantly heals a target with your active Rejuvenation or Regrowth effect for an amount equal to the full duration of the periodic effect of the consumed spell. | same | match | the engine already consumed the effect it reads |
| Nature's Swiftness | Restoration | 1 | When activated, your next Nature spell becomes an instant cast spell. | same | match |  |
| Living Spirit | Restoration | 3 | Increases your Spirit by 15%. | same | match |  |
| Improved Tranquility | Restoration | 2 | Reduces threat caused by Tranquility by 100% and its cooldown by 60%. | 60% cooldown cut | match | the threat half is not modeled |
| Improved Regrowth | Restoration | 5 | Increases the critical effect chance of your Regrowth spell by 50%. | same | match |  |
| Wild Growth | Restoration | 1 | Heals the target and their party for 336 over 7 sec. Party members must be within 43.5 yards of target. The amount healed is applied quickly at first, and slows down a... | same, 3 sec cast | match | the live tooltips give every rank a 3 sec cast; the client rows say instant |

## Hunter

| Talent | Tree | Max | Forever text at max rank | Engine at max rank | Verdict | Note |
|---|---|---|---|---|---|---|
| Deadly Aspects | Beast Mastery | 5 | While Aspect of the Hawk is active, Auto Shot has a 10% chance of increasing ranged attack speed by 30% for 12 sec. While Aspect of the Beast is active, all melee auto... | 10%, 30%, 12 sec | match | Aspect of the Hawk half only; the Aspect of the Beast half has no aspect to key off |
| Endurance Training | Beast Mastery | 5 | Increases the Health and Armor of your pets by 15%. | 15% pet health | match | pet armor not scaled |
| Focused Fire | Beast Mastery | 2 | Increases all damage you and your pet deal by 2% while your pet is active. | same | match |  |
| Improved Aspect of the Monkey | Beast Mastery | 3 | Increases the Dodge bonus of your Aspect of the Monkey by 6%. Additionally, your pet gains 50% of the effect of your Aspect of the Monkey ability. | - | unmodeled | no-op; Aspect of the Monkey is not registered |
| Pathfinding | Beast Mastery | 2 | Increases the speed bonus of your Aspect of the Cheetah and Aspect of the Pack by 6%. | - | unmodeled | movement speed only |
| Improved Revive Pet | Beast Mastery | 2 | Revive Pet's casting time is reduced by 6 sec, mana cost is reduced by 40%, and increases the health your pet returns with by an additional 30%. | - | unmodeled | Revive Pet is never cast |
| Bestial Swiftness | Beast Mastery | 1 | Increases the movement speed of your pets by 30%. | - | unmodeled | pet movement speed only |
| Unleashed Fury | Beast Mastery | 5 | Increases the damage done by your pets and hawks by 15%. | 15% | match (fixed in ef964a5aa) | was 4% a rank |
| Improved Mend Pet | Beast Mastery | 2 | Gives your Mend Pet spell a 50% chance of cleansing 1 Curse, Disease, Magic, or Poison effect from your pet each time it heals and reduces the Mana cost by 20%. | - | unmodeled | Mend Pet is not registered |
| Ferocity | Beast Mastery | 5 | Increases the critical strike chance of your pets and hawks by 10%. | 10% | match (fixed in ef964a5aa) | was 3% a rank |
| Summon Hawk | Beast Mastery | 1 | Command a hawk to dive-bomb your targeted enemy, dealing [32 / Ferocity: 48 / Unleashed Fury: 52 + (Ranged Attack Power * (0.05))] Physical damage and continuing its a... | spell row | text-only | damage is a bracketed template in the text; checked by the spell-row conformance report |
| Spirit Bond | Beast Mastery | 2 | While your pet is active, you and your pet will regenerate 1% of total health every 5 sec. | - | unmodeled | out-of-combat regen tick |
| Intimidation | Beast Mastery | 1 | Command your pet to Stun the target for 3 sec on its next successful attack, which also gains 100% increased critical strike chance. Generates high threat. | - | unmodeled | no pet stun ability |
| Bestial Discipline | Beast Mastery | 2 | Increases the Focus regeneration of your pets by 20% and allows 50% of your Mana regeneration to continue while casting. | 20% focus regen | match | the Mana-regen-while-casting half is not modeled |
| Frenzy | Beast Mastery | 5 | Gives your pet a 100% chance to gain a 30% attack speed increase for 8 sec after dealing a critical strike. | same | match |  |
| Bestial Wrath | Beast Mastery | 1 | Send your pet into a rage causing 50% additional damage for 18 sec. While enraged, the beast does not feel pity or remorse or fear and it cannot be stopped unless killed. | same | match |  |
| Hawk Eye | Marksmanship | 3 | Increases the range of your ranged weapons by 6 yards. | - | unmodeled | range only |
| Improved Concussive Shot | Marksmanship | 5 | Gives your Concussive Shot a 20% chance to stun the target for 3 sec. | - | unmodeled | Concussive Shot not registered |
| Lethal Attacks | Marksmanship | 5 | Increases your critical strike chance with all attacks by 5%. | same | match |  |
| Improved Stings | Marksmanship | 3 | Increases the damage of your Serpent Sting ability by 20%, reduces the cooldown of your Viper Sting ability by 6 sec, and increases the duration of your Scorpid Sting ... | Serpent Sting +20% (6/13/20) | match | Viper Sting cooldown and Scorpid Sting duration: stings not registered |
| Efficiency | Marksmanship | 5 | Reduces the Mana cost of your Shots, Stings, and melee abilities by 15%. | 15% (was 10%) | mismatch, fixed | 3% a rank; was 2% a rank on Shots, Stings and Volley; Strider Kick joins the melee abilities in 1.60.1.70707 (mask word 2 0x2000 -> 0x202000) |
| Careful Aim | Marksmanship | 5 | Increases your Attack Power by 100% of your Intellect. | same | match |  |
| Rapid Killing | Marksmanship | 2 | Reduces the cooldown on your Rapid Fire ability by 2 min. In addition, when you kill a non-trivial enemy or it dies while afflicted by your Serpent Sting, you gain Rap... | 2 min cooldown cut | match | the kill-triggered 20% Shot buff needs a target-death event the engine does not have |
| Improved Arcane Shot | Marksmanship | 5 | Reduces the cooldown of your Arcane Shot by 1.5 sec. Does not affect the cooldown of abilities which share a cooldown with Arcane Shot. | same | match |  |
| Lone Wolf | Marksmanship | 1 | You deal 20% increased damage with all attacks while you do not have an active pet. | same | match |  |
| Trueshot Aura | Marksmanship | 1 | Increases the Ranged Attack Power of party or raid members within 45 yds by 15. | 15/22/30/40/50 ranged AP by trained rank | mismatch, fixed | 1.60.1.70707 (was 30/40/50/75/50); via the shared raid buff (core.TrueshotAuraRanks) |
| Mortal Shots | Marksmanship | 5 | Increases the critical strike damage bonus on all ranged abilities by 30%. | same | match |  |
| Rapid Recuperation | Marksmanship | 2 | Hitting a target with your Serpent Sting ability grants you 50% and consuming Rapid Killing grants you 100% of your Mana regeneration while casting for the next 15 sec. | 50% (Serpent Sting hit), 15 sec | match | the Rapid Killing half cannot fire |
| Barrage | Marksmanship | 3 | Increases the damage done by your Multi-Shot, Aimed Shot, and Volley abilities by 10%. | 10% on Multi-Shot, Aimed Shot, Volley (3/7/10%; was 15% on Multi-Shot only, plus a Volley crit-bonus term) | mismatch, fixed | table 3/7/10, not linear |
| Scatter Shot | Marksmanship | 1 | A short-range shot that deals 50% weapon damage and disorients the target for 4 sec. Any damage caused will remove the effect. Turns off your attack when used. | - | unmodeled | crowd control, not registered |
| Ranged Weapon Specialization | Marksmanship | 5 | Increases the damage you deal with ranged weapons by 5%. | same | match |  |
| Sniper Shot | Marksmanship | 1 | A long-range shot that deals ranged damage plus 160 and increases the range of your next 3 Shots by 10 yards for 10 sec. | 160 at rank 1 | match | spell row, checked by conformance |
| Improved Tracking | Survival | 5 | While tracking Beasts, Demons, Dragonkin, Elementals, Giants, Humanoids, or Undead, all damage you deal to the tracked creature type is increased by 5%. | same | match |  |
| Deflection | Survival | 5 | Increases your Parry chance by 5%. | - | unmodeled | parry chance, defensive |
| Entrapment | Survival | 5 | When your traps are triggered, all affected targets are Entrapped, preventing them from moving for 5 sec. | - | unmodeled | trap root duration |
| Savage Strikes | Survival | 2 | Increases the critical strike chance of all your melee abilities by 4%. | 4% (was 20%) | mismatch, fixed | 2% a rank on Raptor Strike and Mongoose Bite; was 10% a rank. Other melee abilities (Wing Clip, Counterattack, Strider Kick) still carry no bonus |
| Survivalist | Survival | 5 | Increases your total Health by 10%. | same | match |  |
| Improved Wing Clip | Survival | 3 | Gives your Wing Clip ability a 20% chance to immobilize the target for 5 sec. | - | unmodeled | immobilize chance |
| Clever Traps | Survival | 2 | Increases the duration of Freezing and Frost trap effects by 30% and the damage of Immolation and Explosive trap effects by 30%. | +30% trap damage | match | the Freezing/Frost duration half is not modeled |
| Surefooted | Survival | 3 | Increases your hit chance by 3% and reduces the duration of movement impairing effects on you by 30%. | 3% hit | match | movement-impair duration half not modeled |
| Deterrence | Survival | 1 | When activated, increases your Dodge and Parry chance by 25% for 10 sec. | - | unmodeled | not registered |
| Survival Tactics | Survival | 2 | Increases your chance to hit with your Trap and Feign Death abilities by 10%. | - | unmodeled | trap and Feign Death hit chance |
| Predator's Edge | Survival | 5 | Increases your melee critical strike damage by 30% and your Off Hand weapon damage by 50%. | same | match |  |
| Counterattack | Survival | 1 | A strike that becomes active after parrying an opponent's attack. This attack deals 50% weapon damage plus 26 and immobilizes the target for 5 sec. Counterattack canno... | 50% weapon + 26, 5 sec | match | spell row |
| Resourcefulness | Survival | 2 | Reduces the Mana cost of your Trap and melee abilities by 60%, and grants your melee and ranged ability Critical Strikes a 100% chance to allow 50% of your Mana Regeneration to continue while casting for 30 sec. | 50% a rank chance (was 30%) | mismatch, fixed | 1.60.1.70425 reads 50/100% again |
| Expose Prey | Survival | 2 | Your attacks against targets with Hunter's Mark have a 10% chance to activate your Mongoose Bite for 10 sec. | same | match |  |
| Survivalist's Discipline | Survival | 2 | Reduces the cooldown of your Trap and Deterrence abilities by 40%. | same | match |  |
| Strider Kick | Survival | 1 | A powerful kick that deals 100% melee weapon damage and increases movement speed by 30% for 3 sec. | 100% weapon, 30%, 3 sec | match | spell row |
| Lightning Reflexes | Survival | 5 | Increases your Agility by 10%. | same | match |  |
| Lacerating Strikes | Survival | 1 | Your Mongoose Bite also causes the target to Bleed for damage equal to 40% of the damage done by Mongoose Bite over 21 sec | 40% of hit over 21 sec (7 ticks of 3 sec) | match |  |

## Mage

| Talent | Tree | Max | Forever text at max rank | Engine at max rank | Verdict | Note |
|---|---|---|---|---|---|---|
| Wand Specialization | Arcane | 2 | Increases your damage with Wands by 25%. | same | match |  |
| Improved Channeling | Arcane | 5 | Gives your Arcane Missiles spell a 100% chance and your Arcane Blast and Arcane Wrath spells a 70% chance to not lose casting time when you take damage. | - | unmodeled | pushback avoidance never triggers in the fights modeled |
| Arcane Subtlety | Arcane | 2 | Reduces your target's resistance to all your spells by 15 and reduces the threat caused by your Arcane spells by 30%. | 30% less threat | match | the 15 spell-resistance cut is not modeled |
| Magic Absorption | Arcane | 2 | Increases all your resistances by 10 and causes all spells you fully resist to restore 2% of your total mana. Cannot trigger more often than 1 time per sec. | 10 resistance | match | the 2% mana restore on a full resist is not modeled |
| Arcane Concentration | Arcane | 5 | Gives you a 10% chance of entering a Clearcasting state after any damage spell hits a target. The Clearcasting state reduces the mana cost of your next damage spell by... | same | match |  |
| Arcane Resilience | Arcane | 3 | Increases your Armor by an amount equal to 75% of your Intellect. | - | unmodeled | armor from Intellect; three ranks since 1.60.1.70707 |
| Arcane Geometry | Arcane | 2 | Increases the range of your Arcane spells by 6 yards. | - | unmodeled | range only |
| Arcane Impact | Arcane | 3 | Increases the critical strike chance of your Arcane spells by 6%. | same | match |  |
| Arcane Blast | Arcane | 1 | Blasts the target with energy, dealing 34 to 38 Arcane damage. Each time you cast Arcane Blast, you gain 12% of your Intellect as Spell Damage and the mana cost of Arcane Blast is increased by 100%. Effect stacks up to 4 times and lasts 18 sec from time of initial application. | +12% Intellect as spell power a stack, cost +100% a stack, 18 sec from the first stack, not renewed | mismatch, fixed | 1.60.1.70707 hotfix (was 15%, +175%, 15 sec; the shipped 400573 rows still read the old values); 1.60.1.70425 rework |
| Arcane Shielding | Arcane | 2 | Decreases the Mana lost per point of damage taken when your Mana Shield spell is active by 33% and increases the resistances granted by your Mage Armor spell by 50%. | - | unmodeled | Mana Shield and Mage Armor are not modeled |
| Improved Counterspell | Arcane | 2 | Your Counterspell also Silences the target for 4 sec. | - | unmodeled | silence |
| Arcane Meditation | Arcane | 3 | Allows 50% of your Mana regeneration to continue while casting. | same | match |  |
| Missile Barrage | Arcane | 2 | Gives your Arcane Blast spell a 40% chance, and your Arcane Wrath, Fireball, Frostbolt, Frost Blast, and Frostfire Bolt spells a 20% chance to reduce the channeled duration of your next Arcane Missiles spell by 50%, reduce the Mana cost by 100%, and missiles fire every 0.5 sec. | 20/10% a rank | mismatch, fixed | two ranks since 1.60.1.70425; 1.60.1.70532 dropped the Arcane Missile hits' 5% a rank and named Frost Blast, which is not registered (dispositions.go) |
| Arcane Wrath | Arcane | 1 | Launches a projectile of pure Arcane energy at the enemy target, causing 122 to 146 Arcane damage. This spell gains additional Spell Damage equal to 5% of your maximum Mana. | registered under the talent: 3.5 sec cast, 6 sec cooldown, coefficient 1.0, +5% of maximum mana as spell damage | match | new in 1.60.1.70425 (arcane_wrath.go) |
| Presence of Mind | Arcane | 1 | When activated, your next Mage spell with a casting time less than 10 sec becomes an instant cast spell. | same | match |  |
| Arcane Mind | Arcane | 5 | Increases your Intellect by 10% and increases the critical strike damage bonus of your Arcane spells by 100%. | same | match |  |
| Arcane Instability | Arcane | 3 | Increases the damage done by your spells by 3% and your critical strike chance by 3%. | same | match |  |
| Arcane Power | Arcane | 1 | For the next 15 sec, your spells deal 30% more damage while costing 30% more mana to cast. | same | match |  |
| Improved Fire Blast | Fire | 2 | Reduces the cooldown of your Fire Blast spell by 2 sec and increases its range by 15 yards. | Fire Blast cooldown -1 sec a rank | match | Wake of Fire's slot since 1.60.1.70425; range only otherwise |
| Incineration | Fire | 5 | Increases the critical strike chance of your Fire Blast, Ice Lance, Arcane Blast, and Scorch spells by 10%. | 2% a rank | match | five ranks since 1.60.1.70685 (was three) |
| Accelerant | Fire | 5 | Reduces the casting time of your Fireball, Frostfire Bolt, and Arcane Wrath spells by 0.5 sec. | same | match | Improved Fireball renamed in 1.60.1.70685, Arcane Wrath added |
| Ignite | Fire | 5 | Your critical strikes from Fire damage spells cause the target to burn for an additional 40% of your spell's damage over 4 sec. | same | match |  |
| Flame Throwing | Fire | 2 | Increases the range of your Fire spells by 6 yards. | - | unmodeled | range only |
| Impact | Fire | 3 | Gives your Fire spells a 10% chance to stun the target for 2 sec. | - | unmodeled | stun chance |
| Burning Soul | Fire | 3 | Gives your Fire spells a 70% chance to not lose casting time when you take damage and reduces the threat caused by your Fire spells by 30%. | 30% less Fire threat | match | the 70% pushback avoidance is not modeled |
| Improved Flamestrike | Fire | 3 | Increases the critical strike chance of your Flamestrike spell by 15%. | same | match |  |
| Pyroblast | Fire | 1 | Hurls an immense fiery boulder that causes 101 to 131 Fire damage and an additional 44 Fire damage over 12 sec. | same | match |  |
| Improved Scorch | Fire | 3 | Your Scorch spell has a 100% chance to cause your target to be vulnerable to Fire damage. This vulnerability increases all Fire damage you deal to your target by 3% an... | 33/67/100% chance, 3% a stack to 5 stacks, 30 sec (was 33/66/100%) | mismatch, fixed | rank 2 was 66% |
| Improved Fire Ward | Fire | 2 | Causes your Fire Ward to have a 20% chance to reflect Fire spells while active. | - | unmodeled | Fire Ward reflect |
| Heating Up | Fire | 1 | Non-periodic critical strikes with Fireball, Frostfire Bolt, Fire Blast, and Scorch reduce the cast time of your next Pyroblast cast within 20 sec by 25%, stacking up to 3 times. Fire Blast grants an additional stack every time it deals damage. | same; a landed Fire Blast is one stack, a critical one two | match | the Fire Blast clause is 1.60.1.70425's |
| Master of Elements | Fire | 3 | Your Fire and Frost critical strikes will refund 30% of their base mana cost. | same | match |  |
| Critical Mass | Fire | 3 | Increases the critical strike chance of your Fire spells by 6%. | 2% a rank | match | three ranks since 1.60.1.70685 (70425: five); Combustion needs all three |
| Blast Wave | Fire | 1 | A wave of flame radiates outward from the caster, damaging all enemies caught within the blast for 154 to 184 Fire damage, and Dazing them for 50% reduced movement spe... | same | match |  |
| Fire Power | Fire | 3 | Increases the damage done by your Fire spells by 3% per your Fire effects on the target, up to a maximum increase of 12%. Effects which mix multiple schools of magic do not qualify. | 1% a rank per Fire effect on the target (Fireball, Pyroblast and Flamestrike dots, Ignite, Improved Scorch), four effects at most | mismatch, fixed | 1.60.1.70425 rework (fire_power.go); was a flat 2% a rank |
| Combustion | Fire | 1 | When activated, this spell causes each of your Fire damage spell hits to increase your critical strike chance with Fire damage spells by 10%. This effect lasts until y... | same | match |  |
| Frost Warding | Frost | 2 | Increases the Armor and resistances granted by your Frost Armor and Ice Armor spells by 30%, gives your Frost Ward a 20% chance to negate the warded damage spell, and increases the amount absorbed by your Ice Barrier and Frost Ward by 20%. | - | unmodeled | armors, Frost Ward and the absorbs |
| Improved Frostbolt | Frost | 5 | Reduces the casting time of your Frostbolt spell by 0.5 sec. | same | match |  |
| Precision | Frost | 3 | Improves your chance to hit with all spells by 6%. | every mage spell | match | Elemental Precision renamed and widened to every school in 1.60.1.70685; three ranks of 2% since 1.60.1.70532. Spells cast by items carry no mage mask and do not take it |
| Ice Shards | Frost | 5 | Increases the critical strike damage bonus of your Frost spells by 100%. | same | match |  |
| Permafrost | Frost | 3 | Increases the duration of your Chill effects by 33% and reduces the target's speed by an additional 10%. | - | unmodeled | chill duration and slow |
| Improved Frost Nova | Frost | 2 | Reduces the cooldown of your Frost Nova spell by 4 sec. | same | match |  |
| Frostbite | Frost | 3 | Gives your Chill effects a 15% chance to Freeze the target for 5 sec. | same | match |  |
| Frost Channeling | Frost | 3 | Reduces the mana cost of your Frost spells by 15% and reduces the threat caused by your Frost spells by 30%. | 15% cost, 30% threat | match |  |
| Ice Lance | Frost | 1 | Deals 28 to 32 Frost damage to an enemy target. Deals 300% increased damage to Frozen targets. | x4 against a Frozen target | mismatch, fixed | 1.60.1.70707 hotfix: 300% (70425-70685 200%, x3; the shipped row 1342606 still reads 200) |
| Improved Blizzard | Frost | 3 | Adds a Chill effect to your Blizzard spell. This effect lowers the target's movement speed by 40% for 1.5 sec. | chill 1.5 sec | match | the 15/25/40% slow value is not read by the engine; only the chill (Fingers of Frost, Frostbite) is |
| Arctic Reach | Frost | 2 | Increases the range of your Frostbolt and Blizzard spells and the radius of your Frost Nova and Cone of Cold spells by 20%. | - | unmodeled | range and radius |
| Ice Block | Frost | 1 | You become encased in a block of ice, protecting you from all physical attacks and spells for 10 sec, but during that time you cannot attack, move, or cast spells. | - | unmodeled | defensive |
| Shatter | Frost | 3 | Increases the critical strike chance of all your spells against Frozen targets by 50%. | same | match |  |
| Improved Cone of Cold | Frost | 3 | Increases the damage dealt by your Cone of Cold spell by 35%. | same | match |  |
| Cold Snap | Frost | 1 | Finishes the remaining cooldown on all your other Frost spells. | same | match |  |
| Fingers of Frost | Frost | 2 | Gives your Chill effects a 15% chance to grant you the Fingers of Frost effect, which treats your next 2 spells cast as if the target were Frozen. Lasts 15 sec. | same | match |  |
| Winter's Chill | Frost | 5 | Gives your Frostbolt, Frost Blast, and Fingers of Frost empowered Ice Lance spells a 100% chance to apply Winter's Chill, increasing the target's damage taken from your next Cone of Cold by 20% and it gains an additional 7% of your spell damage, stacking up to 5 times. | 20% a rank chance on a landed Frostbolt or Fingers of Frost Ice Lance; 5 stacks; the next Cone of Cold takes +20% damage and +7% of spell damage a stack and spends them | mismatch, fixed | 1.60.1.70707 hotfix (+20% a stack, was +5%); 1.60.1.70532 rework (cone_of_cold.go); Frost Blast is not registered |
| Ice Barrier | Frost | 1 | Shield yourself, absorbing [448 + (Frost spell power * 0.1) * (1)] damage and increasing your Frost damage by 15% for 1 min. While shielded, your Frost damage is increased by an additional 5% and you do not lose casting time when you take damage. | +15% Frost damage for 1 min and +5% while shielded | mismatch, fixed | 1.60.1.70707 hotfix (was 3% + 3%; the shipped rows still read 3 and 3); the absorb is not modeled, nothing hits a simmed mage, so the shield holds its minute (ice_barrier.go) |

## Paladin

| Talent | Tree | Max | Forever text at max rank | Engine at max rank | Verdict | Note |
|---|---|---|---|---|---|---|
| Spiritual Focus | Holy | 5 | Gives your Flash of Light, Holy Light, and Light's Vigil spells a 70% chance to not lose casting time when you take damage. | 70% pushback avoidance (14% a rank) | mismatch, fixed | 1.60.1.70707: five ranks at 14% (was two at 35%); pushback is not rolled in the fights modeled |
| Divine Strength | Holy | 5 | Increases your Strength by 10%. | same | match |  |
| Healing Light | Holy | 3 | Increases the amount healed by your Holy Light, Flash of Light, and Holy Shock spells by 12%. | same | match |  |
| Reverence | Holy | 2 | Allows 30% of your Mana regeneration to continue while casting. | 30% (15% a rank) | mismatch, fixed | 1.60.1.70707: two ranks at 15% (was three at 10%) |
| Improved Seals | Holy | 5 | Increases the damage done by your Seals and Judgements by 15%. | 15% (3% a rank) | mismatch, fixed | 1.60.1.70707: five ranks at 3% (was three at 5%) |
| Purifying Power | Holy | 2 | Reduces the mana cost of your Cleanse and Purify spells by 20% and reduces the cooldown of your Exorcism and Holy Wrath spells by 33%. | same | match |  |
| Divine Favor | Holy | 1 | When activated, gives your next Flash of Light, Holy Light, or Holy Shock spell a 100% critical effect chance. | same | match |  |
| Illumination | Holy | 5 | After getting a critical effect from your Flash of Light, Holy Light, Light's Vigil, or Holy Shock heal spell you have a 100% chance to gain Mana equal to 50% of the b... | same | match |  |
| Holy Conduit | Holy | 2 | Reduces the mana cost of your Consecration, Holy Wrath, Exorcism, and Hammer of Wrath spells by 40%. | same | match | moved from Retribution in 1.60.1.70707 |
| Divine Precision | Holy | 2 | Improves your chance to hit with Holy spells by 8%, and reduces the duration of Silence effects on you by 20%. | 8% Holy spell hit (4% a rank) | mismatch, fixed | 1.60.1.70707: two ranks at 4% (was three at 6%); the Silence duration half is not modeled |
| Infusion of Light | Holy | 2 | Your Holy Shock and Flash of Light critical hits reduce the cast time of your next Holy Light cast within 15 sec by 1.0 sec. | same | match |  |
| Oath of Light | Holy | 1 | Swear an Oath to protect target ally for 5 min, increasing the healing of your Holy Light on them by up to 0, your Flash of Light by up to 0, and your Holy Shock by up... | - | unmodeled | new in 1.60.1.70707; the text reads "by up to 0" for every heal and its spell (1328548) is not in the client tables |
| Consecrated Ground | Holy | 2 | Gives your Holy spells 10% increased damage against the first 4 enemies that enter your Consecration. | same | match |  |
| Holy Shock | Holy | 1 | Blasts the target with Holy energy, causing 129 to 139 Holy damage to an enemy, or 110 to 118 healing to an ally. | same | match |  |
| Boundless Light | Holy | 2 | Increases the critical strike chance of your Holy Shock spell by 30% when cast on allies not affected by your Oath of Light. | 30% Holy Shock heal crit | match | new in 1.60.1.70707; Oath of Light is never sworn, so every ally qualifies |
| Holy Power | Holy | 5 | Increases the critical strike healing bonus of your Holy spells by 50%. | heal crit bonus x1.5 (150% heals crit for 175%) | mismatch, fixed | 1.60.1.70707: crit healing bonus, scaling the bonus as Ruin does (was crit chance) |
| Light's Vigil | Holy | 1 | Applies Light's Vigil to a friendly target for 30 sec. The caster's next Holy Shock on the target incurs no cooldown and heals the target and their party for 326 to 34... | same | match | 1.60.1.70707 dropped the enemy half; the engine only ever modeled the heal |
| Divine Intellect | Protection | 5 | Increases your total Intellect by 10%. | same | match | moved from Holy in 1.60.1.70707 |
| Redoubt | Protection | 5 | Damaging melee attacks against you have a 10% chance to increase your chance to block by 30%. Lasts 10 sec or 5 blocks. | 30% block (6% a rank) | mismatch, fixed | 1.60.1.70707 (was 4% a rank) |
| Precision | Protection | 3 | Improves your chance to hit by 3%. | same | match |  |
| Guardian's Favor | Protection | 2 | Reduces the cooldown of your Blessing of Protection by 2 min and increases the duration of your Blessing of Freedom by 6 sec. | - | unmodeled | Blessing of Protection cooldown and Freedom duration |
| Sacred Duty | Protection | 2 | Reduces the cooldown of your Divine Shield, Divine Protection, and Templar's Bulwark spells by 60 sec, and reduces the duration of Forbearance by 10 sec. | -60 sec cooldowns, Forbearance -10 sec | mismatch, fixed | 1.60.1.70707: the 4% Stamina is gone, Forbearance is 10 sec shorter |
| Templar's Bulwark | Protection | 1 | When activated, this ability grants you an absorb shield equal to 100% of your maximum health for 8 sec. Applies Forbearance for 1 min. Cannot be cast while Forbearanc... | same | match |  |
| Improved Hammer of Justice | Protection | 3 | Decreases the cooldown of your Hammer of Justice spell by 15 sec. | - | unmodeled | Hammer of Justice cooldown |
| Shield Specialization | Protection | 2 | Increases the amount of damage absorbed by your shield by 30%, and gives your blocks a 100% chance to restore 4% of your maximum Mana. May only occur once every 3 sec. | 30% block value, 100% chance at 4% maximum mana | mismatch, fixed | 1.60.1.70707: two ranks at 15% and 50% (was three at 10% and 33%) |
| Toughness | Protection | 5 | Increases your armor value from items by 10%. | same | match |  |
| Swift Judgement | Protection | 1 | Unleash the energy of a Seal spell upon an enemy. Does not consume the Seal. Refer to individual Seals for Judgement effect. Does not share a cooldown with your Judgem... | a second Judgement of the active Seal, 1 min cooldown, no cost | mismatch, fixed | 1.60.1.70707 (was a Judgement cooldown reset and a free next Judgement); cooldown and cost from the client row and the live tooltip |
| Improved Righteous Fury | Protection | 3 | While Righteous Fury is active, all damage taken is reduced by 9%. | 9% (3% a rank) | mismatch, fixed | 1.60.1.70707 (was 2% a rank) |
| Anticipation | Protection | 5 | Increases your Defense Skill by 20. | same | match |  |
| Stoicism | Protection | 2 | Reduces the duration of Stun and Charm effects on you by 20%. | - | unmodeled | new in 1.60.1.70707: Stun and Charm duration |
| Holy Shield | Protection | 1 | Increases chance to block by 30% for 10 sec, and deals 75 Holy damage for each attack blocked while active. Damage caused by Holy Shield causes 20% additional threat. ... | 75/110/153 by rank at levels 30/40/50 | mismatch, fixed | 1.60.1.70707 hotfix moved every rank one bracket down (Wowhead live tooltips); the client rows still read 110/153/221 at 40/50/60 |
| Reckoning | Protection | 5 | Gives you a 40% chance after Blocking and a 100% chance after receiving a non-periodic critical strike to gain an extra attack. Stores up to 4 extra attacks. Cannot oc... | same chances | match | the engine grants the extra attack at once, so the four-attack store is never reached; the 1.5 sec limit is applied to both triggers (text: blocks only) |
| Iron Creed | Protection | 3 | Increases the threat generated by your Holy Strike ability by 30%. While Righteous Fury is active, Holy Strike also reduces your damage taken by 6% for 6 sec. | 30% Holy Strike threat, 6% damage taken for 6 sec | mismatch, fixed | 1.60.1.70707: three ranks at 10% and 2% (was five at 5% and 2%); needs the Righteous Fury option |
| Shieldsworn | Protection | 1 | Reduces the cooldown of your Holy Strike ability by 4 sec. Additionally, Holy Strike now strikes with your shield, triggering your active Holy Shield and Shield Specia... | Holy Strike cooldown -4 sec; the strike fires Holy Shield damage and Shield Specialization mana without charges or cooldown | match | new in 1.60.1.70707; needs a shield; the 6 sec spell block is not modeled |
| Deflection | Retribution | 5 | Increases your Parry chance by 5%. | same | match |  |
| Benediction | Retribution | 5 | Reduces the Mana cost of all instant cast spells and abilities by 10%. | same | match |  |
| Improved Judgement | Retribution | 2 | Decreases the cooldown of your Judgement ability by 2 sec. | same | match |  |
| Conviction | Retribution | 3 | Improves your chance to get a critical strike with all spells and attacks by 6%. | 6% crit (2% a rank) | mismatch, fixed | 1.60.1.70707: three ranks at 2%, all spells and attacks (was five at 1%, melee); Forever has one crit stat |
| Sanctified Judgement | Retribution | 3 | Gives your Judgement ability a 100% chance to return 60% of the Mana cost of the judged seal. | same | match |  |
| Vindication | Retribution | 3 | Gives your damaging melee attacks a chance to reduce the target's Attack Power by (3 /- 3 * 6) for 30 sec. | - | unmodeled | 1.60.1.70707 removed the self attack power buff the engine modeled; the target Attack Power cut is a garbled template ("(3 /- 3 * 6)") |
| Unyielding Faith | Retribution | 2 | Reduces the duration of Fear and Disorient effects on you by 30%. | - | unmodeled | moved from Holy in 1.60.1.70707; Fear and Disorient duration |
| Seal of Command | Retribution | 1 | Gives the Paladin a chance to deal additional Holy damage equal to 70% of normal weapon damage. Only one Seal can be active on the Paladin at any one time. Lasts 30 se... | same | match |  |
| Pursuit of Justice | Retribution | 2 | Increases movement speed and mounted movement speed by 15%. This does not stack with other movement speed increasing effects. | - | unmodeled | movement speed |
| Eye for an Eye | Retribution | 2 | All critical strikes against you cause 10% of the damage taken to the attacker as well. The damage caused by Eye for an Eye will not exceed 50% of the Paladin's total ... | - | unmodeled | reflected damage |
| Sacred Arbiter | Retribution | 1 | Increases the damage of your Holy Strike ability by 20% and causes it to refresh all Judgement effects on the target. | same | match |  |
| Two-Handed Weapon Specialization | Retribution | 3 | Increases the physical damage you deal by 6%, and reduces all threat you generate by 20% while a two-handed melee weapon is equipped. | 6% physical damage, 20% less threat, with a two-hander | mismatch, fixed | 1.60.1.70707: physical damage only, so seal procs, Judgement of Command, Holy Strike and Hammer of the Righteous lose it; the two-hander condition is read as covering the damage too |
| Turn Evil | Retribution | 2 | Reduces the cast time of your Turn Undead spell by 1.0 sec, its cooldown by 10 sec, and allows it to affect Demons. | - | unmodeled | new in 1.60.1.70707: Turn Undead |
| Repentance | Retribution | 1 | Puts the enemy target in a state of meditation, incapacitating them for up to 6 sec. Any damage caused will awaken the target. Only works against Humanoids. | - | unmodeled | crowd control |
| Vengeance | Retribution | 3 | Increases your Physical and Holy damage dealt by 3% for 30 sec after landing a non-periodic critical strike. Stacks up to 3 times. | 3% a stack, 3 stacks, 30 sec = 9% | match |  |
| Champion of the Light | Retribution | 5 | Increases your spell damage by up to 60% of your Intellect, and reduces the cast time of your Hammer of Wrath by 1.0 sec. | 60% of Intellect, Hammer of Wrath cast -1.0 sec | mismatch, fixed | 1.60.1.70707: five ranks at 12% and 0.2 sec (was three at 20%, no cast time) |
| Unsealed Wrath | Retribution | 1 | Your melee attacks store a copy of your active Seal for 15 sec, stacking up to 3 times. Landing an attack with a different Seal consumes all stacks, giving the stored ... | stores the Seal on each white swing (3 stacks, 15 sec); a swing under another Seal rolls the stored Seal once a stack | match | new in 1.60.1.70707; reading in unsealed_wrath.go (white swings only; Seal of Command's 1 sec cooldown not applied to stored copies) |

## Priest

| Talent | Tree | Max | Forever text at max rank | Engine at max rank | Verdict | Note |
|---|---|---|---|---|---|---|
| Power in Light | Discipline | 5 | Your Smite and Penance spells deal 10% increased damage to targets afflicted with your Holy Fire or Shadow Word: Pain. | - | unmodeled | the Smite/Penance loop is not a damage spec the sim ranks |
| Wand Specialization | Discipline | 2 | Increases your damage with Wands by 25%. | 25% wand damage | match |  |
| Twin Disciplines | Discipline | 5 | Increases the damage and healing of your instant cast spells by 5%. | same | match |  |
| Silent Resolve | Discipline | 3 | Reduces the threat generated by your Holy spells by 30% and reduces the duration of Stun, Fear, and Silence effects inflicted on you by 15%. | 30% less Holy threat (was 12% less of all threat) | mismatch, fixed | vanilla 4% a rank on all spells; stun/fear/silence duration half not modeled |
| Holy Precision | Discipline | 3 | Improves your chance to hit with Holy spells by 18%. | - | unmodeled | Holy spell hit chance is not applied |
| Improved Power Word: Shield | Discipline | 3 | Increases the damage absorbed by your Power Word: Shield by 20%. | same | match |  |
| Martyrdom | Discipline | 2 | Gives you a 100% chance to gain Focused Casting for 6 sec after being the victim of a melee or ranged critical strike. The Focused Casting effect prevents you from los... | - | unmodeled | defensive |
| Mental Agility | Discipline | 3 | Reduces the mana cost of your Smite, Holy Fire, and instant cast spells by 10%. | 10% cost | match | Smite, Holy Fire and instant casts |
| Inner Focus | Discipline | 1 | When activated, reduces the Mana cost of your next spell by 100% and increases its critical effect chance by 25% if it is a non-periodic spell and capable of a critica... | same | match |  |
| Meditation | Discipline | 3 | Allows 50% of your Mana regeneration to continue while casting. | same | match |  |
| Improved Inner Fire | Discipline | 3 | Increases the Armor bonus of your Inner Fire spell by 45% and increases its total charges by 12. | armor +45% | match | the +4/+8/+12 charges are not modeled |
| Mental Strength | Discipline | 5 | Increases your total Intellect by 10%. | 2% a rank | mismatch, fixed | 1.60.1.70425 (was 15%, 3% a rank) |
| Soul Warding | Discipline | 1 | Reduces the cooldown on your Power Word: Shield spell by 4 sec and reduces its mana cost by 15%. | same | match |  |
| Improved Mana Burn | Discipline | 2 | Reduces the casting time of your Mana Burn spell by 1.0 sec. | - | unmodeled | Mana Burn not registered |
| Penance | Discipline | 1 | Launches a volley of holy light at the target, causing 45 Holy damage to an enemy, or 134 healing to an ally, instantly and every 1 sec for 2 sec. | spell row | text-only | checked by the spell-row conformance report |
| Renewed Hope | Discipline | 5 | Your heals from Flash Heal, Lesser Heal, Heal, Greater Heal, and Penance gain 10% increased critical strike chance when cast on targets with Weakened Soul, and reduce the remaining duration of Weakened Soul on their target by 5 sec. | 10% crit on Weakened Soul targets, -5 sec Weakened Soul | match | Binding Heal left the list with its talent in 1.60.1.70707 |
| Divine Aegis | Discipline | 3 | Your non-periodic critical heals create a protective shield on the target, absorbing 15% of the amount healed. Shields up to (20 * 60) damage and lasts 12 sec. | 5% a rank, capped at 20 a level | match | the cap is 1.60.1.70425's |
| Power Infusion | Discipline | 1 | Infuses the target with power, increasing their spell damage and healing done by 20% for 15 sec. | same | match |  |
| Twilight Focus | Holy | 3 | Gives your spells a 70% chance to not lose casting time when you take damage. | - | unmodeled | pushback avoidance |
| Improved Renew | Holy | 3 | Increases the amount healed by your Renew spell by 15%. | same | match |  |
| Holy Specialization | Holy | 5 | Increases the critical effect chance of your Holy spells by 5%. | same | match |  |
| Spell Warding | Holy | 5 | Reduces all spell damage taken by 10%. | same | match |  |
| Divine Fury | Holy | 5 | Reduces the casting time of your Smite, Holy Fire, Heal, and Greater Heal spells by 0.5 sec. | same | match |  |
| Holy Nova | Holy | 1 | Causes an explosion of holy light around the caster, causing 26 to 30 Holy damage to all enemy targets within 10 yards and healing all party members within 10 yards fo... | same | match |  |
| Blessed Recovery | Holy | 3 | After being struck by a melee or ranged critical hit, or suffering more than 30% of your maximum Health from a single attack, heal 25% of the damage taken over 6 sec. ... | - | unmodeled | defensive heal |
| Inspiration | Holy | 3 | Your non-periodic critical heals increase your target's Armor by 25% for 15 sec. | same | match |  |
| Holy Reach | Holy | 2 | Increases the range of your Smite and Holy Fire spells and the radius of your Prayer of Healing and Holy Nova spells by 20%. | - | unmodeled | range and radius |
| Improved Healing | Holy | 3 | Reduces the Mana cost of your Lesser Heal, Heal, Greater Heal, Penance, and Prayer of Mending spells by 15%. | 15% cost | match | Penance and Prayer of Mending included |
| Searing Light | Holy | 2 | Increases your Holy damage done by 5%, and gives a 10% chance each time your Holy Fire spell deals periodic damage for your next Holy Nova to cost no Mana. | Holy damage +2/5% (was 5/10%) | mismatch, fixed | Smite and Holy Fire only; the Holy Nova free-cast chance is not modeled |
| Holy Binding | Holy | 1 | Each time you heal a target other than yourself with Flash Heal, you are healed for 50% as much. This healing generates no threat. | - | unmodeled | 1.60.1.70707, Binding Heal's replacement; the raid damage model never damages the priest, so the self-heal is all overhealing |
| Litany of Light | Holy | 2 | When you cast a healing spell, gain Mana equal to 10% of the base cost of the spell if your previous heal was a different spell. | same | match |  |
| Spirit of Redemption | Holy | 1 | Upon death, the priest becomes the Spirit of Redemption for 15 sec. The Spirit of Redemption cannot move, attack, be attacked or targeted by any spells or effects. Whi... | - | unmodeled | on death |
| Spiritual Guidance | Holy | 5 | Increases your spell healing by up to 25% of your total Spirit and your spell damage by up to 8% of your total Spirit. | healing 25% of Spirit | match | the spell-damage half (1/3/5/6/8% of Spirit) is not modeled; the engine adds only the healing dependency |
| Spiritual Healing | Holy | 3 | Increases the amount healed by your spells by 10%. | same | match |  |
| Prayer of Mending | Holy | 1 | Places a spell on the target that heals them for [(172 + (Healing * 0.42899999)) * (1 * 1)] the next time they take damage or receive non-periodic healing. When the he... | spell row | text-only | the text is a bracketed template; checked by the spell-row conformance report |
| Shadow Focus | Shadow | 5 | Improves your chance to hit with Shadow spells by 5%. | 5% Shadow hit (was 10% on every priest spell) | mismatch, fixed | 2% a rank on every priest spell |
| Blackout | Shadow | 5 | Gives your Shadow damage spells a 10% chance to stun the target for 3 sec. | - | unmodeled | stun chance |
| Spirit Tap | Shadow | 5 | Gives you a 100% chance to increase your Spirit by 100% for 15 sec after killing a non-trivial target or when an enemy afflicted by your Vampiric Embrace dies. For the... | - | unmodeled | the aura is registered but nothing triggers it: it needs a target-death event |
| Improved Shadow Word: Pain | Shadow | 2 | Increases the duration of your Shadow Word: Pain spell by 6 sec. | same | match |  |
| Shadow Reach | Shadow | 2 | Increases the range of your offensive Shadow spells by 20%. | - | unmodeled | range |
| Improved Mind Blast | Shadow | 5 | Reduces the cooldown of your Mind Blast spell by 2.5 sec. | same | match | moved to tier 1 (Shadow Affinity's slot) in 1.60.1.70425; the tier-2 node is gone |
| Improved Psychic Scream | Shadow | 2 | Reduces the cooldown of your Psychic Scream spell by 4 sec. | - | unmodeled | cooldown of a spell not registered |
| Mind Flay | Shadow | 1 | Assault the target's mind with Shadow energy, causing 63 Shadow damage over 3 sec and slowing their movement speed by 50%. | same | match |  |
| Improved Mind Flay | Shadow | 2 | Each instance of damage of Mind Flay deals 20% more damage than its previous instance of damage. | tick n is (1 + 10% a rank)^(n-1) of the snapshot | mismatch, fixed | 1.60.1.70425 rework (mind_flay.go); was a flat +10% a rank |
| Dark Inspiration | Shadow | 5 | Your Mind Blast and fully channeled Mind Flay causes Dark Inspiration, increasing yours and your party's Spirit by 15% for 10 sec. | 3% a rank Spirit on the priest for 10 sec, from a landed Mind Blast or the third Mind Flay tick | match | Improved Fade's slot since 1.60.1.70425; the party share is not modeled |
| Vampiric Embrace | Shadow | 1 | Afflicts your target with Shadow energy that causes all party members to be healed for 20% of any Shadow spell damage you deal for 30 sec. | 20% of Shadow damage to the party | match |  |
| Shadow Weaving | Shadow | 3 | Your Shadow damage spells have a 100% chance to increase the Shadow damage you deal by 2% for 15 sec, stacking up to 5 times. | same | match |  |
| Silence | Shadow | 1 | Silences the target, preventing them from casting spells for 5 sec and interrupting their spellcasts for 3 sec. | - | unmodeled | interrupt |
| Devouring Contagion | Shadow | 2 | Reduces the mana cost of your Devouring Plague by 50%. Targets that die while Devouring Plague is active spreads it, jumping to a nearby enemy within 10 yds for the re... | Devouring Plague cost -50% | match | the spread-on-death half needs a target-death event |
| Early Demise | Shadow | 2 | Increases Shadow Word: Death's critical strike chance on targets at or below 20% health by 30%. | same | match |  |
| Darkness | Shadow | 5 | Increases your Shadow damage done by 10%. | same | match |  |
| Shadowform | Shadow | 1 | Assume Shadowform, increasing your Shadow damage by 10%, reducing the Mana cost of all Shadow spells by 50%, increasing the critical strike damage bonus of your Shadow... | Shadow +10%, Shadow cost -50%, Shadow crit damage bonus +100%, Physical damage taken -15% (was +15% Shadow damage only) | mismatch, fixed |  |

## Rogue

| Talent | Tree | Max | Forever text at max rank | Engine at max rank | Verdict | Note |
|---|---|---|---|---|---|---|
| Improved Gouge | Assassination | 3 | Increases the duration of your Gouge ability by 1.5 sec. | - | unmodeled | Gouge duration (stun-like crowd control) |
| Remorseless Attacks | Assassination | 2 | After killing a non-trivial enemy, gives you a 40% increased critical strike chance on your next Sinister Strike, Backstab, Ambush, Mutilate, or Ghostly Strike. Lasts ... | - | unmodeled | kill-triggered buff needs a target-death event |
| Malice | Assassination | 5 | Increases your critical strike chance with all attacks and Poisons by 5%. | 5% crit | match | the "and Poisons" half is not separately applied |
| Ruthlessness | Assassination | 3 | Gives your finishing moves a 60% chance to add a Combo Point to your target. | same | match |  |
| Murder | Assassination | 2 | Increases all damage dealt by 4% against Humanoid and Giant targets. | same | match |  |
| Improved Slice and Dice | Assassination | 3 | Increases the duration of your Slice and Dice ability by 45%. | same | match |  |
| Relentless Strikes | Assassination | 1 | Your finishing moves have a 20% chance per Combo Point to restore 25 Energy. | same | match |  |
| Improved Expose Armor | Assassination | 2 | Reduces the Energy cost of your Expose Armor ability by 10, and refunds 2 Combo Points when cast with 5 Combo Points. | same | match |  |
| Lethality | Assassination | 5 | Increases the critical strike damage bonus of your Sinister Strike, Gouge, Backstab, Mutilate, Ghostly Strike, and Hemorrhage abilities by 20%. | same | match |  |
| Vile Poisons | Assassination | 5 | Increases the damage dealt by your poisons by 20% and gives your poisons an additional 40% chance to resist dispel effects. | +20% poison damage | match | the dispel resist half is not modeled |
| Cold Blood | Assassination | 1 | When activated, increases the critical strike chance of your next Sinister Strike, Backstab, Ambush, Eviscerate, or Mutilate by 100%. | same | match |  |
| Improved Poisons | Assassination | 5 | Increases the chance to apply Poisons to your target by 10%, and gives Poison applications a 50% chance to not consume a charge. | 10% apply chance | match | the 50% no-charge-consumed half is not modeled |
| Vigor | Assassination | 2 | Increases your maximum Energy by 10. | same | match |  |
| Mutilate | Assassination | 1 | Instantly attacks with both weapons for 75% weapon damage plus an additional 17 with each weapon. Damage increased by 20% against Poisoned targets. Awards 2 Combo Points. | same | match |  |
| Improved Kidney Shot | Assassination | 2 | Enemies Stunned by your Kidney Shot ability take 10% increased damage from your poisons and attacks. | - | unmodeled | Kidney Shot not registered |
| Seal Fate | Assassination | 5 | Your critical strikes from abilities that add Combo Points have a 100% chance to add an additional Combo Point. | 100% extra combo point | match | the engine adds a 0.5 sec internal cooldown the text does not state |
| Venom | Assassination | 1 | Finishing move that increases the damage of your Poisons by 30% and your chance to apply Poisons by 10%. Lasts longer per combo point: 1 point : 9 sec 2 points: 12 sec... | +30% poison damage, +10% apply chance, 9 to 21 sec | match |  |
| Improved Eviscerate | Combat | 3 | Increases the damage done by your Eviscerate ability by 20%. | same | match |  |
| Improved Sinister Strike | Combat | 2 | Reduces the Energy cost of your Sinister Strike ability by 5. | same | match |  |
| Lightning Reflexes | Combat | 5 | Increases your Dodge chance by 5%. | same | match |  |
| Puncturing Wounds | Combat | 3 | Increases the critical strike chance of your Backstab by 30% and your Mutilate by 15%, and gives Backstab a 45% chance to add an additional Combo Point. | Backstab +30% crit, Mutilate +15%, 45% extra combo point | match |  |
| Deflection | Combat | 3 | Increases your Parry chance by 6%. | 6% parry (was 3%) | mismatch, fixed | 1% a rank before |
| Precision | Combat | 3 | Improves your chance to hit by 3%. | same | match |  |
| Endurance | Combat | 2 | Reduces the cooldown of your Sprint and Evasion abilities by 60%. | - | unmodeled | Sprint and Evasion cooldown; defensive |
| Riposte | Combat | 1 | A strike that becomes active after parrying an opponent's attack. This attack deals 150% weapon damage and disarms the target for 6 sec. | 150% weapon damage, 6 sec disarm | match | spell row; the disarm is not modeled |
| Improved Sprint | Combat | 2 | Gives a 100% chance to remove all movement impairing effects when you activate your Sprint ability. | - | unmodeled | Sprint not registered |
| Improved Kick | Combat | 2 | Gives your Kick ability a 100% chance to Silence the target for 2 sec. | - | unmodeled | silence |
| Flawless Execution | Combat | 1 | Reduces the Energy cost of your Eviscerate ability by 10. | same | match |  |
| Dual Wield Specialization | Combat | 5 | Increases the damage done by your off-hand weapon by 25%. | same | match |  |
| Blade Flurry | Combat | 1 | Increases your melee attack speed by 20% and your melee attacks strike an additional nearby opponent. Lasts 15 sec. | same | match |  |
| Hack and Slash | Combat | 5 | Gives your melee weapon attacks a benefit depending on the weapon. Axe/Sword: Your successful melee attacks have a 5% chance to trigger an extra attack on the target. ... | 5% extra attack (Axe/Sword), 5% crit (Dagger/Fist), 15% armor penetration (Mace) | match |  |
| Weapon Expertise | Combat | 2 | Reduces the chance for your attacks to be Dodged or Parried by 2%. | same | match |  |
| Aggression | Combat | 3 | Increases the damage of your Sinister Strike, Backstab, and Eviscerate abilities by 6%. | same | match |  |
| Adrenaline Rush | Combat | 1 | Increases your Energy regeneration rate by 100% for 15 sec. | same | match |  |
| Camouflage | Subtlety | 5 | Reduces your speed penalty from your Stealth ability by 15% and reduces its cooldown by 6 sec. | - | unmodeled | Stealth |
| Master of Deception | Subtlety | 3 | Reduces the chance enemies have to detect you while in Stealth mode as if you were 3 levels higher. | - | unmodeled | stealth detection |
| Opportunity | Subtlety | 2 | Increases the damage dealt by your Backstab, Garrote, Ambush, and Mutilate abilities by 10%. | same | match |  |
| Setup | Subtlety | 3 | Gives you a 100% chance to add a Combo Point to your target after Dodging one of their attacks or fully resisting one of their spells. | 100% combo point | match | 33/67/100 |
| Elusiveness | Subtlety | 2 | Reduces the cooldown of your Vanish and Blind abilities by 90 sec. | Vanish cooldown -90 sec; Evasion no longer cut (rank-2 Evasion cooldown was 5 sec - 90 sec, negative) | mismatch, fixed | Blind not registered. Evasion is only reachable through RegisterEvasionSpell, which nothing calls |
| Dirty Tricks | Subtlety | 2 | Reduces the Energy cost of your Sap and Blind abilities by 50%. | - | unmodeled | Sap and Blind not registered |
| Improved Ambush | Subtlety | 3 | Increases the critical strike chance of your Ambush ability by 45%. | same | match |  |
| Initiative | Subtlety | 3 | Gives you a 100% chance to add an additional combo point to your target when using your Ambush, Garrote, or Cheap Shot ability. | 100% combo point | match | 33/67/100 |
| Ghostly Strike | Subtlety | 1 | A strike that deals 125% (180% if a Dagger is equipped in your Main Hand) weapon damage and increases your chance to dodge by 15% for 7 sec. Awards 1 combo points. | same | match |  |
| Improved Distract | Subtlety | 2 | Increases the radius of your Distract ability by 5 yds, and further reduces the Stealth detection of distracted enemies as though they were an additional 2 levels lower. | - | unmodeled | Distract |
| Heightened Senses | Subtlety | 2 | Increases your Stealth detection as if you were 3 levels higher and reduces your chance to be hit by spells and ranged attacks by 4%. | - | unmodeled | stealth detection and spell/ranged hit avoidance |
| Premeditation | Subtlety | 1 | Adds 2 Combo Points to your target. You must add to or use those combo points within 20 sec or the combo points are lost. | same | match |  |
| Serrated Blades | Subtlety | 3 | Causes your attacks to ignore 9% of your target's Armor and increases the damage dealt by your Rupture ability by 30%. | 9% armor ignored, Rupture +30% | match |  |
| Dirty Deeds | Subtlety | 2 | Reduces the Energy cost of your Cheap Shot and Garrote abilities by 20, and your Garrote ability no longer requires you to be behind your target. | same | match |  |
| Preparation | Subtlety | 1 | When activated, this ability immediately finishes the cooldown on your other Rogue abilities. | same | match |  |
| Hemorrhage | Subtlety | 1 | An instant strike that deals 100% weapon damage (145% if a Dagger is equipped) and causes the target to take 15% increased Rupture damage from the Rogue. Lasts 15 sec.... | same | match |  |
| Quietus | Subtlety | 5 | Your Sinister Strike, Ghostly Strike, and Hemorrhage abilities cause 10% more damage against targets below 35% health. | same | match |  |
| Cutthroat | Subtlety | 5 | Your Backstab has a 15% chance to cause your next Ambush within 10 sec to not require Stealth. | same | match |  |
| Thousand Cuts | Subtlety | 1 | When your Rupture ability deals periodic damage, the Energy cost of your next Hemorrhage or Backstab ability within 10 sec is reduced by 3, stacking up to 5 times. | same | match |  |

## Shaman

| Talent | Tree | Max | Forever text at max rank | Engine at max rank | Verdict | Note |
|---|---|---|---|---|---|---|
| Convection | Elemental | 5 | Reduces the mana cost of your Shock, Lightning Bolt, Lava Burst, and Chain Lightning spells by 10%. | same | match |  |
| Concussion | Elemental | 5 | Increases the damage done by your Lightning Bolt, Chain Lightning, and Earth Shock spells by 5%. | same | match |  |
| Elemental Warding | Elemental | 3 | Reduces damage taken from Fire, Frost, and Nature effects by 10%. | - | unmodeled | defensive |
| Reverberation | Elemental | 5 | Reduces the cooldown of your Shock spells by 1.0 sec. | same | match |  |
| Call of Flame | Elemental | 3 | Increases the damage done by your Fire Totems and by your Flame Shock, Fire Nova, and Lava Burst spells by 15%, as well as increases the duration of your Flame Shock spell by 9 sec. | 5% a rank; Flame Shock +1 tick (3 sec) a rank | match | the duration clause is 1.60.1.70425's |
| Elemental Devastation | Elemental | 3 | Your offensive spell critical strikes will increase your chance to get a critical strike with melee attacks by 9% for 10 sec. | same | match |  |
| Elemental Focus | Elemental | 1 | Gives you a 10% chance to enter a Clearcasting state after casting any Fire, Frost, or Nature damage spell. The Clearcasting state reduces the mana cost of your next d... | same | match |  |
| Elemental Alacrity | Elemental | 3 | Reduces the cast time of your Lightning Bolt, Chain Lightning, and Lava Burst spells by 0.50 sec. | same | match |  |
| Trial by Fire | Elemental | 2 | Increases the damage done by your Fire Nova spell by 20% and reduces its cooldown by 4 sec, and increases the duration of your Magma Totem by 20 sec. | 10% and 2 sec a rank; Magma Totem +10 sec a rank | match | Improved Fire Nova's slot since 1.60.1.70425 |
| Eye of the Storm | Elemental | 3 | Gives your Lightning Bolt, Chain Lightning, and Lava Burst spells a 70% chance to not lose casting time when you take damage. | 70% pushback reduction | match | 1.60.1.70532 words it as a chance; pushback is not rolled in the fights modeled |
| Elemental Reach | Elemental | 2 | Increases the range of your Lightning Bolt, Chain Lightning, Fire Nova, and Lava Burst spells by 6 yards and reduces the cooldown of Totemic Projection by 20 sec. | - | unmodeled | range, and a totem-move cooldown the sim never casts |
| Lightning Overload | Elemental | 3 | Gives your Lightning Bolt and Chain Lightning spells a 10% chance to cast a second, similar spell on the same target at no additional cost that causes half damage and ... | same | match |  |
| Earthbound | Elemental | 1 | Your Earthbind Totem Immobilizes nearby targets for 5 sec when cast. | - | unmodeled | crowd control |
| Elemental Fury | Elemental | 5 | Increases the critical strike damage bonus of your Searing and Magma Totems and your Fire, Frost, and Nature spells by 100%. | +100% crit damage bonus | match |  |
| Lava Burst | Elemental | 1 | You hurl molten lava at the target, dealing 114 to 145 Fire damage. If your Flame Shock is on the target, Lava Burst deals 20% increased damage. | spellconst rows (regenerated for 1.60.1.70425) | match | moved to tier 4 in 1.60.1.70425 |
| Earth's Grasp | Enhancement | 2 | Increases the health of your Stoneclaw Totem by 50% and the radius of your Earthbind Totem by 20%. | - | unmodeled | totem health and radius |
| Thundering Strikes | Enhancement | 5 | Improves your chance to get a critical strike with all spells and attacks by 5%. | same | match |  |
| Ancestral Knowledge | Enhancement | 5 | Increases your Intellect by 10%. | +10% Intellect (was +5% maximum Mana) | mismatch, fixed | 1% max Mana a rank before |
| Guardian Totems | Enhancement | 2 | Increases the amount of damage reduced by your Stoneskin Totem and Windwall Totem by 20% and reduces the cooldown of your Grounding Totem by 2 sec. | Stoneskin +20% | match | Windwall and Grounding Totem cooldown not modeled |
| Mental Dexterity | Enhancement | 3 | Increases your Attack Power by an amount equal to 100% of your Intellect. | same | match |  |
| Improved Ghost Wolf | Enhancement | 2 | Reduces the cast time of your Ghost Wolf spell by 3.0 sec, and Ghost Wolf may be used indoors. | - | unmodeled | travel |
| Improved Lightning Shield | Enhancement | 3 | Increases the damage done by your Lightning Shield orbs by 15%. | same | match |  |
| Elemental Weapons | Enhancement | 3 | Increases the melee attack power bonus of your Rockbiter Weapon by 20%, your Windfury Weapon effect by 40% and increases the damage caused by your Flametongue Weapon a... | Rockbiter +20%, Windfury +40%, Flametongue/Frostbrand +15% | mismatch, fixed | Rockbiter rank 2 was 14% (text 13%) |
| Shamanistic Focus | Enhancement | 1 | Reduces the mana cost of your Shock and Lightning Shield spells by 45%. | same | match |  |
| Anticipation | Enhancement | 3 | Increases your chance to dodge by an additional 6%. | 6% dodge (was 3%) | mismatch, fixed | 1% a rank before |
| Toughness | Enhancement | 5 | Increases your Stamina by 10%. | +10% Stamina (was +10% armor from items) | mismatch, fixed | target stat was wrong |
| Flurry | Enhancement | 5 | Increases your attack speed by 25% for your next 3 swings after dealing a melee critical strike. | +25% attack speed (was +30%); +5% at 1/5 (was +10%) | mismatch, fixed | vanilla 10/15/20/25/30 |
| Stormstrike | Enhancement | 1 | Instantly strike for normal weapon damage and increase the damage you deal to the target with your next Lightning Bolt, Chain Lightning, or Earth Shock spell by 20% fo... | +20% for 12 sec | match | spell row |
| Spirit Weapons | Enhancement | 1 | Gives a chance to parry enemy melee attacks, reduces all threat generated by your attacks by 30% while Rockbiter Weapon is not active, and increases all threat generat... | parry | match | the threat halves are not modeled |
| Mental Quickness | Enhancement | 2 | Increases your spell damage and healing by up to 30% of your Intellect. | same | match |  |
| Improved Stormstrike | Enhancement | 2 | When you Stormstrike, you have a 100% chance to gain 50% mana regeneration while casting spells for 15 sec, and Stormstrike's cooldown has a 100% chance to reset each ... | 100% mana regen proc, 100% cooldown reset on dodge/parry | match |  |
| Maelstrom Weapon | Enhancement | 5 | When you deal damage with a melee attack, you have a chance to reduce the cast time and Mana cost of your next Lightning Bolt spell by 20%. Stacks up to 5 times. Lasts... | 20% per stack, 5 stacks, 30 sec | match |  |
| Rage of the Farseer | Enhancement | 1 | Increases your attack speed by 30% for 25 sec. | same | match |  |
| Improved Healing Wave | Restoration | 5 | Reduces the casting time of your Healing Wave spell by 0.5 sec. | same | match |  |
| Totemic Focus | Restoration | 5 | Reduces the Mana cost of your totems and any spells that summon or move them by 25%. | same | match |  |
| Mindfulness | Restoration | 3 | Allows 50% of your Mana regeneration to continue while casting. | same | match |  |
| Natural Grace | Restoration | 3 | Reduces the threat generated by your spells by 15%. | same | match |  |
| Tidal Focus | Restoration | 5 | Reduces the Mana cost of your healing spells by 5% and improves your chance to hit by 5%. | same | match |  |
| Improved Reincarnation | Restoration | 2 | Reduces the cooldown of your Reincarnation spell by 20 min, increases your maximum health by 4%, and increases the amount of health and Mana you reincarnate with by an... | - | unmodeled | no death in the sim |
| Ancestral Healing | Restoration | 3 | Increases your target's armor value by 25% for 15 sec after getting a critical effect from one of your healing spells. | - | unmodeled | armor on healed target |
| Healing Focus | Restoration | 3 | Gives your healing spells a 70% chance to not lose casting time when you take damage. | - | unmodeled | pushback avoidance |
| Water Shield | Restoration | 1 | The caster is surrounded by 3 globes of water. When a spell, melee, or ranged attack hits the caster or when one of the caster's spells gets a critical result, 2% of maximum mana is restored to the caster, expending one water globe. Only one globe will activate every few seconds.  Lasts 10 min. | a critical heal or a critical damage spell spends a globe | match | 1.60.1.70425 widened the crit clause from healing spells to spells |
| Tidal Mastery | Restoration | 5 | Increases the critical effect chance of your healing spells by 5%. | same | match |  |
| Restorative Totems | Restoration | 5 | Increases the effect of your Mana Spring Totem by 25% and increases the effect of your Healing Stream Totem by 50%. | same | match |  |
| Mana Tide Totem | Restoration | 1 | Summons a Mana Tide Totem with 5 health at the feet of the caster for 12 sec that restores 88 mana every 3 seconds to group members within 30 yards. | spell row | text-only | checked by the spell-row conformance report |
| Healing Way | Restoration | 3 | Increases the amount healed by your Healing Wave spell by 25%. | same | match |  |
| Nature's Swiftness | Restoration | 1 | When activated, your next Nature spell with a casting time less than 10 sec. becomes an instant cast spell. | same | match |  |
| Purification | Restoration | 5 | Increases the effectiveness of your healing spells by 10%. | same | match |  |
| Riptide | Restoration | 1 | Heals a friendly target for 486 to 534, an additional 445 over 15 sec, and increases the effectiveness of your Chain Heal casts directly on that target by 25%. | spell row | text-only | checked by the spell-row conformance report |

## Warlock

| Talent | Tree | Max | Forever text at max rank | Engine at max rank | Verdict | Note |
|---|---|---|---|---|---|---|
| Improved Life Tap | Affliction | 2 | Increases the amount of Mana awarded by your Life Tap spell by 20%. | same | match |  |
| Suppression | Affliction | 5 | Improves your chance to hit by 5% and reduces all threat you generate by 20%. | same | match |  |
| Improved Corruption | Affliction | 5 | Reduces the casting time of your Corruption spell by 2 sec and increases the damage it deals by 10%. | -2 sec cast, +10% damage (was cast time only) | mismatch, fixed | damage half was unread |
| Grim Reach | Affliction | 2 | Increases the range of your Affliction spells by 20%. | - | unmodeled | Malediction's slot since 1.60.1.70425; range only |
| Soul Harvest | Affliction | 2 | Killing a non-trivial target afflicted by your Drain Soul increases your Mana regeneration by 100% for 10 sec and allows 100% of normal Mana regeneration to continue w... | - | unmodeled | kill-triggered regen needs a target-death event |
| Improved Drains | Affliction | 3 | Increases Health drained, damage done, or Mana drained by your Drain Life, Drain Soul, and Drain Mana spells by 20%. | 20% Drain Life and Drain Soul damage | mismatch, fixed | 1.60.1.70707 names Drain Mana where it named Wrack; Wrack no longer gains it (the shipped mask 403511 still carries Wrack); Drain Mana is not registered |
| Improved Bane of Agony | Affliction | 2 | Increases the damage done by your Bane of Agony by 10%. | same | match |  |
| Fel Concentration | Affliction | 3 | Gives your Drain Life, Drain Mana, Drain Soul, and Wrack spells a 70% chance to not lose casting time when you take damage. | 70% pushback reduction | match | 1.60.1.70532 words it as a chance; pushback is not rolled in the fights modeled |
| Amplify Curse | Affliction | 1 | Increases the effect of your next Curse of Weakness or Bane of Agony by 50%, or your next Curse of Exhaustion by 20%. Lasts 30 sec. | +50% Curse of Weakness / Bane of Agony, +20% Exhaustion | match | spell |
| Pandemic | Affliction | 3 | Increases the critical strike damage bonus of your Corruption, Bane of Agony, Bane of Doom, Drain Soul, Drain Life, Siphon Life, and Wrack spells by 100%. | same | match |  |
| Malevolence | Affliction | 5 | Increases the critical effect chance of your Shadow spells by 5%. | same | match |  |
| Nightfall | Affliction | 2 | Gives your Corruption, Drain Soul, Drain Life, and Wrack spells a 4% chance to cause you to enter a Shadow Trance after damaging the opponent. The Shadow Trance reduce... | 4% on Corruption and Drain Life | match | Drain Soul and Wrack procs are not wired |
| Curse of Exhaustion | Affliction | 1 | Reduces the target's movement speed by 30% for 12 sec. Only one Curse per Warlock can be active on any one target. | - | unmodeled | movement speed |
| Siphon Life | Affliction | 1 | Transfers 11 health from the target to the caster every 3 sec. Lasts 30 sec. | same | match |  |
| Soul Siphon | Affliction | 3 | Increases the damage done or health drained by your Drain Life, Drain Soul, and Wrack spells by 12% per each of your other Affliction effects active on the target, up ... | 12% per effect, up to 36% | match |  |
| Shadow Mastery | Affliction | 5 | Increases the damage dealt or life drained by your Shadow spells by 10%. | 2% a rank | mismatch, fixed | 1.60.1.70425 (was 5%, 1% a rank) |
| Wrack | Affliction | 1 | Tears the target apart from within, inflicting 36 Shadow damage every 1 sec and increasing the damage they take from your other Shadow damage over time effects by 10% ... | same | match |  |
| Improved Health Funnel | Demonology | 2 | Increases the amount of health transferred by your Health Funnel spell by 40%, reduces its health cost by 30%, and reduces all threat your Health Funnel generates by 1... | - | unmodeled | Health Funnel not registered |
| Improved Imp | Demonology | 3 | Increases the damage of your Imp's Firebolt spell by 30% and the effect of its Fire Shield spell by 30%. | Firebolt +30% | match | the Fire Shield half is not modeled |
| Demonic Embrace | Demonology | 5 | Increases your total Stamina by 15%. | +15% Stamina, no Spirit change (was -5% Spirit) | mismatch, fixed | vanilla Spirit penalty removed |
| Unholy Power | Demonology | 5 | Increases all damage done by your Imp, Voidwalker, Succubus, Incubus, and Felhunter pets by 10%. | +10% pet damage (was +20% on pet melee only) | mismatch, fixed | 2% a rank on every demon damage source (pet-wide multiplier; Firebolt included, round 3) |
| Demonic Aegis | Demonology | 2 | Increases the effectiveness of your Demon Skin and Demon Armor spells by 30%. | - | unmodeled | armor spells |
| Improved Voidwalker | Demonology | 3 | Increases the effectiveness of your Voidwalker's Torment, Consume Shadows, Sacrifice, and Suffering spells by 30%. | - | unmodeled | Voidwalker utility spells |
| Fel Vitality | Demonology | 3 | Increases the maximum health and Mana of your Imp, Voidwalker, Succubus, Incubus, and Felhunter by 15%, and increases your maximum Mana by 15%. | same | match |  |
| Demonic Energies | Demonology | 2 | You heal your pet for 15% of all spell damage you deal. When you gain Mana from Life Tap, your summoned demon gains 100% of the Mana you gain. | 100% of Life Tap mana to the demon | match | the 15% spell-damage heal to the pet is not modeled |
| Improved Sayaad | Demonology | 3 | Increases the effect of your Succubus' and Incubus' Lash of Pain and Soothing Kiss spells by 30%, and increases the duration of your Succubus' and Incubus' Seduction a... | Lash of Pain +30% | match | the Seduction/Invisibility duration half is not modeled |
| Demonic Sacrifice | Demonology | 1 | When activated, sacrifices your summoned Demon to enhance the opposing aspect of your power, granting you an effect that lasts 2 hrs. The effect is canceled if any Dem... | Imp +15% Shadow, Succubus +15% Fire, Voidwalker 2% Mana / 4 sec, Felhunter 3% Health / 4 sec | mismatch, fixed | the four effects were swapped (Imp Fire, Succubus Shadow, Voidwalker Health, Felhunter Mana) |
| Master Summoner | Demonology | 2 | Reduces the casting time of your Imp, Voidwalker, Succubus, Incubus, and Felhunter Summoning spells by 4 sec and the Mana cost by 40%. | -4 sec cast, -40% cost | match |  |
| Decimation | Demonology | 2 | Reduces the cooldown of your Soul Fire spell by 90%. When you cast Shadow Bolt or Searing Pain on an enemy below 35% health, they deal 6% increased damage, and for the... | cooldown -90%, +6% damage, cast -40% | match | "costs no Soul Shards": Soul Fire costs Mana in this fork, so the engine cuts the Mana cost 40% instead |
| Fel Domination | Demonology | 1 | Your next Imp, Voidwalker, Succubus, Incubus, or Felhunter Summon spell has its casting time reduced by 6 sec and its Mana cost reduced by 50%. | -6 sec cast, -50% cost | match | spell |
| Demonic Brand | Demonology | 3 | Your Searing Pain generates 50% less threat and brands the target for 10 sec. Your pet's next 6 attacks against the target deal ((((60 - 26) * 1.5) + 14 + (0.078 * ((S... | - | match | sim round 21: threat 17/33/50%, 2/4/6 charges (curves 83195, 103069), each a ((level - 26) x 1.5 + 14 to 17) + 0.078 x school spell power hit, Shadow (1293697) or the Imp's Fire (1293698); the amount is the talent text and the Wowhead tooltip, one source |
| Improved Felhunter | Demonology | 3 | Increases the Attack Power reduction of your Felhunter's Tainted Blood, the healing of its Devour Magic, and the detection level of its Paranoia by 30%, and reduces th... | - | unmodeled | Felhunter utility spells |
| Soul Link | Demonology | 1 | When active, 30% of all damage taken by the caster is taken by your Imp, Voidwalker, Succubus, Incubus, or Felhunter Demon instead. In addition, both the Demon and the... | caster takes 70% (was 76.9%), +3% damage both | mismatch, fixed | was /1.3 |
| Demonic Knowledge | Demonology | 3 | Increases your spell damage and your Demon pet's spell damage by up to 100% of your level while you have a summoned Demon pet active. | same | match |  |
| Master Demonologist | Demonology | 5 | Grants both the Warlock and the summoned demon an effect as long as that demon is active. Imp - Increases Fire damage done by 10%. Voidwalker - Reduces Physical damage... | Imp +10% Fire, Voidwalker -10% Physical taken, Succubus +10% Shadow, Felhunter -10% Magic taken | mismatch, fixed | was Imp -20% threat, VW -10% all damage taken, Succubus +10% all damage, Felhunter +10 resistance |
| Demonic Pact | Demonology | 1 | Your Demonic Sacrifice effect is no longer cancelled by summoning a different Demon pet. Resummoning the sacrificed pet will still cancel the effect. | same | match |  |
| Destructive Reach | Destruction | 2 | Increases the range of your damaging spells by 20%. | - | unmodeled | range |
| Improved Shadow Bolt | Destruction | 5 | Your Shadow Bolt critical strikes increase Shadow damage taken by the target from your attacks by 20% for 12 sec. | +20% Shadow damage taken for 12 sec, no charge limit (was 4 charges) | mismatch, fixed | the debuff (17794) has SpellDuration 12000 ms and no SpellAuraOptions row, so no proc charges; the bonus lasts 12 sec however many spells land |
| Bane | Destruction | 5 | Reduces the casting time of your Shadow Bolt, Immolate, and Incinerate spells by 0.5 sec and your Soul Fire spell by 2 sec. | -0.5 sec Shadow Bolt, Immolate, Incinerate; -2 sec Soul Fire | mismatch, fixed | Incinerate was not cut |
| Molten Skin | Destruction | 5 | Reduces all damage taken by 10%. | - | unmodeled | defensive |
| Cataclysm | Destruction | 3 | Reduces the Mana cost of your Destruction spells by 10%. | same | match |  |
| Aftermath | Destruction | 5 | Increases the initial damage of your Immolate spell by 50% and your Conflagrate spell has a 100% chance to Daze the target, reducing the target's movement speed by 50%... | Immolate initial +50% | match | the Conflagrate daze is not modeled |
| Ruin | Destruction | 5 | Increases the critical strike damage bonus of your Destruction spells by 100%. | +100% Destruction crit damage bonus, 20% a rank (was +100% at any rank) | mismatch, fixed | below 5/5 it overstated |
| Shadowburn | Destruction | 1 | Instantly blasts the target for 65 to 74 Shadow damage. If a non-trivial target dies within 8 sec of being hit with Shadowburn, the caster gains a Soul Shard. | same | match |  |
| Intensity | Destruction | 3 | Gives your Destruction spells a 70% chance to not lose casting time when you take damage. | 70% pushback reduction | match | 1.60.1.70532 words it as a chance; pushback is not rolled in the fights modeled |
| Agonizing Flames | Destruction | 3 | Increases the critical strike chance of your Searing Pain spell by 10% and the damage done by all your Destruction spells by 10%. | same | match |  |
| Conflagrate | Destruction | 1 | Ignites a target that is already afflicted by your Immolate spell, dealing 88 to 111 Fire damage and consuming your Immolate effect. | same | match |  |
| Pyroclasm | Destruction | 2 | Gives your Soul Fire spell a 26% chance to Stun the target for 3 sec, and your Rain of Fire and Hellfire spells a 26% chance over their duration to Stun targets they d... | - | unmodeled | stun chance |
| Bane of Havoc | Destruction | 1 | Afflicts the target for 5 min, causing 15% of all damage done by the Warlock to other targets to also be dealt to the cursed target. Bane of Havoc is limited to 1 targ... | same | match |  |
| Fire and Brimstone | Destruction | 3 | Increases the critical strike chance of your Conflagrate spell by 25%. | same | match |  |
| Shadow and Flame | Destruction | 5 | Hitting an enemy with Conflagrate increases all Shadow damage you deal by 10% for 20 sec, and hitting an enemy with Shadowburn increases all Fire damage you deal by 10... | same | match |  |
| Incinerate | Destruction | 1 | Deals 100 to 114 Fire damage to your target and an additional 25% damage if the target is afflicted by Immolate. | same | match |  |

## Warrior

| Talent | Tree | Max | Forever text at max rank | Engine at max rank | Verdict | Note |
|---|---|---|---|---|---|---|
| Improved Heroic Strike | Arms | 3 | Reduces the cost of your Heroic Strike ability by 3 Rage. | same | match |  |
| Deflection | Arms | 5 | Increases your Parry chance by 5%. | same | match |  |
| Improved Rend | Arms | 3 | Increases the Bleed damage done by your Rend ability by 35%. | same | match |  |
| Improved Charge | Arms | 2 | Increases the Rage generated by your Charge ability by 6. | same | match |  |
| Improved Tactical Mastery | Arms | 5 | Tactical Mastery lets you retain up to an additional 15 Rage when you change stances. | retain 15 Rage | match |  |
| Improved Overpower | Arms | 2 | Increases the critical strike chance of your Overpower ability by 50%. | same | match |  |
| Anger Management | Arms | 1 | Generates 1 Rage every 3 sec while in combat, and reduces Rage loss while out of combat by 30%. | same | match |  |
| Deep Wounds | Arms | 3 | Your critical strikes cause your opponent to Bleed, dealing 60% of your melee weapon's average damage over 12 sec. | 60% of weapon average damage over 12 sec | match |  |
| Spearing Strike | Arms | 1 | A brutal attack that deals 40% weapon damage. Deals an additional 80% weapon damage against Giants, Dragonkin, and mounted targets. Mounted targets are dismounted. | same | match |  |
| Two-Handed Weapon Specialization | Arms | 3 | Increases the damage you deal with two-handed melee weapons by 3%. | same | match |  |
| Impale | Arms | 2 | Increases the critical strike damage bonus of your abilities by 20%. | same | match |  |
| Bloodthrill | Arms | 5 | Your Main Hand melee attacks against enemies afflicted by your Rend have a 20% chance to allow the use of your Overpower ability on the target. Lasts 6 sec. | 20% chance, 6 sec window | match |  |
| Sweeping Strikes | Arms | 1 | Your next 5 melee attacks strike an additional nearby opponent. | same | match |  |
| Weaponmaster | Arms | 5 | Gives your melee weapon attacks a benefit depending on the weapon. Axe/Polearm: Increases your critical strike chance by 5%. Mace/Staff: Your attacks ignore 15% of you... | 5% crit (Axe/Polearm), 15% armor ignored (Mace/Staff), 5% extra attack (Sword) | match |  |
| Improved Slam | Arms | 2 | Reduces the global cooldown and cast time of your Slam ability by 0.50 sec. In addition, Slam no longer interrupts or delays your melee swing and Slam's cooldown is re... | cast and GCD -0.5 sec; cooldown -3 sec a rank = -6 sec at 2/2 | unsettled | Reading A (the 70291 text): "Slam's cooldown is reduced by 3.0 sec" at both ranks, 3 sec flat; Reading B (the engine, from Blizzard's 1 October 2026 notes "3 s off the cooldown per rank"): 6 sec at 2/2. The client rows (spell 12862, one spell for both ranks, cooldown base -3000 ms, cast time -500 ms) do not tell the readings apart; engine left as is |
| Improved Hamstring | Arms | 3 | Gives your Hamstring ability a 15% chance to immobilize the target for 5 sec. | - | unmodeled | immobilize chance |
| Mortal Strike | Arms | 1 | A vicious strike that deals weapon damage plus 85 and wounds the target, reducing the effectiveness of any healing by 50% for 10 sec. | same | match |  |
| Booming Voice | Fury | 5 | Increases the area of effect of your Shouts by 50% and reduces their Rage cost by 25%. | same | match |  |
| Cruelty | Fury | 5 | Improves your chance to get a critical strike with melee attacks by 5%. | same | match |  |
| Lingering Rage | Fury | 5 | Increases the time before your Rage begins to decay after leaving combat by 10 sec. | - | unmodeled | out-of-combat rage decay |
| Unbridled Wrath | Fury | 5 | Gives you a 60% chance to generate 1 additional Rage when you deal melee damage with a weapon. | same | match |  |
| Furious Precision | Fury | 3 | Increases your chance to hit with off-hand attacks by 10%. | same | match |  |
| Piercing Howl | Fury | 1 | Causes all enemies within 10 yds to be Dazed, reducing movement speed by 50% for 6 sec. | same | match |  |
| Blood Craze | Fury | 3 | Regenerates 3% of your total Health over 6 sec after being the victim of a critical strike or suffering more than 20% of your maximum Health from a single attack. | same | match |  |
| Dual Wield Specialization | Fury | 5 | Increases the damage done by your off-hand weapon by 25% and the Rage generated by your off-hand attacks by 50%. | same | match |  |
| Raging Blows | Fury | 1 | Reduces the Rage cost of your Cleave and Whirlwind abilities by 3. | same | match |  |
| Enrage | Fury | 5 | Gives you a 30% chance to deal 10% increased Physical damage for 12 sec after being the victim of any damaging attack. | 30% chance, +10% Physical damage, 12 sec, on any damaging hit taken (was 100% on a melee crit taken, 12 attack charges) | mismatch, fixed | the chance was an unresolved template, the engine kept vanilla's trigger |
| Improved Execute | Fury | 2 | Reduces the Rage cost of your Execute ability by 5. | same | match |  |
| Improved Berserker Rage | Fury | 2 | Your Berserker Rage ability will instantly generate 10 Rage and has a 100% chance to remove all movement impairing effects when activated. | 10 Rage | match | the movement-impair removal chance is not modeled |
| Death Wish | Fury | 1 | When activated, increases your Physical damage done by 20% and makes you immune to Fear effects, but increases all damage you take by 5%. Lasts 30 sec. | +20% Physical damage, +5% damage taken, 30 sec | match | spell |
| Improved Intercept | Fury | 2 | Reduces the cooldown of your Intercept ability by 10 sec. | - | unmodeled | Intercept cooldown |
| Flurry | Fury | 5 | Increases your melee attack speed by 25% for your next 3 swings after dealing a melee critical strike. | same | match |  |
| Gore Drinker | Fury | 2 | Your Enrage, Berserker Rage, Bloodrage, Death Wish, and Bloodthirst abilities cause your next 3 melee attacks to restore 1.0% of your maximum Health. | - | unmodeled | health restore |
| Bloodthirst | Fury | 1 | Instantly attack the target causing damage equal to 45% of your Attack Power plus 30 and increasing your movement speed by 10% for 10 sec. | same | match |  |
| Improved Bloodrage | Protection | 2 | Increases all the Rage generated by your Bloodrage ability by 50%. | same | match |  |
| Shield Specialization | Protection | 5 | Increases your chance to Block attacks with your shield by 5% and grants you a 100% chance to generate 5 Rage when you Block. | same | match |  |
| Anticipation | Protection | 5 | Increases your Defense Skill by 20. | same | match |  |
| Improved Revenge | Protection | 3 | Increases damage dealt by your Revenge ability by 60%. | same | match |  |
| Improved Thunder Clap | Protection | 3 | Reduces the Rage cost of your Thunder Clap ability by 6. | same | match |  |
| Last Stand | Protection | 1 | When activated, this ability temporarily grants you 30% of your maximum health for 20 sec. After the effect expires, the health is lost. | +30% maximum health, 20 sec | match | spell |
| Master of Defense | Protection | 2 | Grants you a 100% chance to generate 5 Rage when you Dodge or Parry while a shield is equipped. | same | match |  |
| Defiance | Protection | 3 | Increases all threat generated in Defensive stance by an additional 15% while a shield is equipped. | same | match |  |
| Improved Sunder Armor | Protection | 3 | Reduces the Rage cost of your Sunder Armor ability by 3. | same | match |  |
| Improved Shield Wall | Protection | 2 | Reduces the cooldown of your Shield Wall ability by 11.0 min. | cooldown -11.0 min | match |  |
| Focused Rage | Protection | 3 | Reduces the Rage cost of your offensive abilities by 3. | same | match |  |
| Bastion | Protection | 5 | Increases all damage you deal by 10% while a shield is equipped. | +10% Physical damage with a shield | match | text says all damage; the engine applies Physical only |
| Shield Slam | Protection | 1 | Slam the target with your shield, causing 421 to 439 damage, increased by your Block Value, and has a 50% chance of dispelling 1 magic effect on the target. Causes a v... | same | match |  |
| Iron Will | Protection | 5 | Reduces the duration of Stun and Fear effects inflicted on you by 15%. | - | not read | field is not read |
| Improved Disarm | Protection | 3 | Reduces the cooldown of your Disarm ability by 20 secs. | - | not read | field is not read |
| Vanguard | Protection | 1 | Your Charge ability is now usable while in Defensive Stance. | - | not read | field is not read |
| Improved Shield Bash | Protection | 2 | Gives your Shield Bash ability a 100% chance to Silence the target for 3 sec. | - | not read | field is not read |
| Concussion Blow | Protection | 1 | Stuns the target for 5 sec. | - | not read | field is not read |
