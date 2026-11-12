package hunter

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// FOREVER: the client's trait trees replaced vanilla's, so some of the
// talents this file reaches for no longer exist under these names, and
// some changed their rank count and so their proto type. Their behaviour
// is rewritten when this spec is brought up, in rankings-population order
// (design section 2.3). Every site is commented rather than deleted, so
// the diff shows a reviewer exactly what the old tree did, and each is
// left reading the value an untalented character would have read - which
// is what a talent nobody can now take is worth.

// Pet talent rates per rank, Forever's (1.60.1.70291 talent text):
// Ferocity 2% pet and hawk crit a rank (10% at 5/5), Unleashed Fury 3% pet
// and hawk damage a rank (15% at 5/5).
const (
	ferocityCritPerRank  = 2.0
	unleashedFuryPerRank = 0.03
)

// Forever's rates for three more talents (1.60.1.70291 talent text):
// Efficiency "Reduces the Mana cost of your Shots, Stings, and melee
// abilities by 15%" at 5/5 (3% a rank), Barrage (19461) 5/10/15% Multi-Shot,
// Aimed Shot and Volley damage by effect 0's TraitDefinitionEffectPoints
// curve (the 70753 hotfix push; Wowhead's text read 3/7/10% until 70755)
// and Savage Strikes "Increases the critical
// strike chance of all your melee abilities by 4%" at 2/2 (2% a rank).
// Vanilla's 2%, 5% a rank and 10% a rank are not Forever's.
const (
	efficiencyCostReductionPerRank int32   = 3
	savageStrikesCritPerRank       float64 = 2.0
)

// lightningReflexesAgilityPerRank is Lightning Reflexes (19168) effect 0,
// aura 137 (total stat percent): 3/6/9/12/15% Agility by its
// TraitDefinitionEffectPoints curve since the 70753 hotfix push (2% a rank
// before; Wowhead's text caught up in 70755).
const lightningReflexesAgilityPerRank = 0.03

var barrageDamageByRank = [4]float64{0, 0.05, 0.10, 0.15}

// barrageDamage is Barrage's damage bonus at a rank.
func barrageDamage(rank int32) float64 {
	return barrageDamageByRank[rankIndex(rank, barrageDamageByRank[:])]
}

// barrageDamageMultiplier is the damage multiplier Barrage gives Multi-Shot,
// Aimed Shot and Volley.
func (hunter *Hunter) barrageDamageMultiplier() float64 {
	return 1 + barrageDamage(hunter.Talents.Barrage)
}

// savageStrikesCritRating is the bonus crit rating Savage Strikes gives a
// melee ability: every spell its class mask names (66 / 528384 / 2097152),
// Raptor Strike, Mongoose Bite, Wing Clip, Counterattack, Lacerate and
// Strider Kick.
func (hunter *Hunter) savageStrikesCritRating() float64 {
	return savageStrikesCritPerRank * float64(hunter.Talents.SavageStrikes) * core.CritRatingPerCritChance
}

func (hunter *Hunter) ApplyTalents() {
	hunter.registerHeartOfTheLion()

	if hunter.pet != nil {
		hunter.applyFrenzy()
		hunter.registerBestialWrathCD()

		// Forever's rates, from the 1.60.1.70291 talent text (talents/
		// hunter.json): Ferocity "critical strike chance of your pets and
		// hawks by 10%" at 5/5, Unleashed Fury "damage done by your pets and
		// hawks by 15%" at 5/5. Vanilla's were 3% and 4% a rank; the engine
		// carried those until 2026-10-09. One write under the unified Crit.
		hunter.pet.AddStat(stats.Crit, core.CritRatingPerCritChance*ferocityCritPerRank*float64(hunter.Talents.Ferocity))

		// The client's Unleashed Fury adds points to the pet's Tamed Pet
		// Passive (aura 107 on 8875), so it adds with Orc Command's +5.
		hunter.pet.AddPassiveDamagePercent(100 * unleashedFuryPerRank * float64(hunter.Talents.UnleashedFury))

		if hunter.Talents.EnduranceTraining > 0 {
			hunter.pet.MultiplyStat(stats.Health, 1+(0.03*float64(hunter.Talents.EnduranceTraining)))
		}
	}

	/*
		if hunter.Talents.MonsterSlaying+hunter.Talents.HumanoidSlaying > 0 {
			hunter.Env.RegisterPostFinalizeEffect(func() {
				for _, t := range hunter.Env.Encounter.Targets {
					switch t.MobType {
					case proto.MobType_MobTypeHumanoid:
						multiplier := []float64{1, 1.01, 1.02, 1.03}[hunter.Talents.HumanoidSlaying]
						for _, at := range hunter.AttackTables[t.UnitIndex] {
							at.DamageDealtMultiplier *= multiplier
							at.CritMultiplier *= multiplier
						}
					case proto.MobType_MobTypeBeast, proto.MobType_MobTypeGiant, proto.MobType_MobTypeDragonkin:
						multiplier := []float64{1, 1.01, 1.02, 1.03}[hunter.Talents.MonsterSlaying]
						for _, at := range hunter.AttackTables[t.UnitIndex] {
							at.DamageDealtMultiplier *= multiplier
							at.CritMultiplier *= multiplier
						}
					}
				}
			})
		}
	*/

	if hunter.Talents.BestialDiscipline > 0 {
		// 19590's second effect is aura 134 (mana regeneration continues
		// while casting) at 25 a rank: "allows 50% of your Mana Regeneration
		// to continue while casting" at 2/2.
		hunter.PseudoStats.SpiritRegenRateCasting += bestialDisciplineRegenPerRank * float64(hunter.Talents.BestialDiscipline)
		core.MakePermanent(hunter.RegisterAura(core.Aura{
			Label: "Bestial Discipline",
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				if hunter.pet != nil {
					hunter.pet.AddFocusRegenMultiplier(0.1 * float64(hunter.Talents.BestialDiscipline))
				}
			},
		}))
	}

	// Forever: merged from MeleeHit 1/rank + SpellHit 1/rank. One effect
	// under a unified stat gets one write.
	hunter.AddStat(stats.Hit, float64(hunter.Talents.Surefooted)*1*core.HitRatingPerHitChance)

	/*
		hunter.AddStat(stats.Crit, float64(hunter.Talents.KillerInstinct)*1*core.CritRatingPerCritChance)
	*/

	/*
		if hunter.Talents.LethalShots > 0 {
			lethalBonus := 1 * float64(hunter.Talents.LethalShots) * core.CritRatingPerCritChance
			hunter.OnSpellRegistered(func(spell *core.Spell) {
				if spell.Flags.Matches(SpellFlagShot) {
					spell.BonusCritRating += lethalBonus
				}
			})
			hunter.AutoAttacks.RangedConfig().BonusCritRating += lethalBonus
		}
	*/

	if hunter.Talents.RangedWeaponSpecialization > 0 {
		mult := 1 + 0.01*float64(hunter.Talents.RangedWeaponSpecialization)
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			// The hawk's dive is not a ranged weapon attack.
			if spell.ProcMask.Matches(core.ProcMaskRanged) && spell.SpellCode != SpellCode_HunterSerpentSting && spell.SpellCode != SpellCode_HunterSummonHawk {
				spell.DamageMultiplier *= mult
			}
		})
	}

	if hunter.Talents.Survivalist > 0 {
		hunter.MultiplyStat(stats.Health, 1.0+0.02*float64(hunter.Talents.Survivalist))
	}

	if hunter.Talents.LightningReflexes > 0 {
		agiBonus := lightningReflexesAgilityPerRank * float64(hunter.Talents.LightningReflexes)
		hunter.MultiplyStat(stats.Agility, 1.0+agiBonus)
	}

	hunter.applyPredatorsEdge()

	hunter.applyEfficiency()
	hunter.applyTrapMastery()
	hunter.applyCleverTraps()

	// Beast Mastery
	hunter.applyDeadlyAspects()
	hunter.applyFocusedFire()

	// Marksmanship
	hunter.applyLethalAttacks()
	hunter.applyImprovedStings()
	hunter.applyCarefulAim()
	hunter.applyRapidKilling()
	hunter.applyLoneWolf()
	hunter.applyRapidRecuperation()

	// Survival
	hunter.applyImprovedTracking()
	hunter.applyResourcefulness()
	hunter.applyExposePrey()
	hunter.applySurvivalistsDiscipline()

	hunter.applyUnmodeledTalents()
}

// applyUnmodeledTalents is every proto.HunterTalents field this package
// deliberately does not read: each changes something this engine has no
// number for (range, a pet-only survivability/speed/revival stat, a
// crowd-control duration or chance, a defensive cooldown) rather than
// damage, crit, hit, haste, attack power, pet damage, mana for the
// rotation, a proc, or a shot/strike modifier. `t` makes each line a
// one-to-one match against proto/hunter.proto's field list rather than
// a `hunter.Talents.` repeated thirty times.
func (hunter *Hunter) applyUnmodeledTalents() {
	t := hunter.Talents

	// Beast Mastery
	_ = t.ImprovedAspectOfTheMonkey // Dodge bonus (and sharing 50% of it to the pet); Aspect of the Monkey itself is not registered in this package.
	_ = t.Pathfinding               // Movement speed bonus to Aspect of the Cheetah/Pack; no movement-speed number affects a stationary Patchwerk-style DPS sim.
	_ = t.ImprovedRevivePet         // Revive Pet cast time/cost/health; pets never die in this sim (no target-vs-pet combat), so Revive Pet is never cast.
	_ = t.BestialSwiftness          // Pet movement speed only.
	_ = t.ImprovedMendPet           // Mend Pet cleanse chance and mana cost; Mend Pet (a heal) is not registered as a spell here.
	_ = t.SpiritBond                // Out-of-combat-adjacent %-of-health regen tick; not a damage, crit, hit, haste, AP, or mana-for-the-rotation number.
	_ = t.Intimidation              // Pet stun + a one-attack crit buff; the pet has no stun ability registered in pet_abilities.go to grant it.

	// Marksmanship
	_ = t.HawkEye                // Ranged weapon range; no range mechanic bounds a hunter's DPS rotation in this engine.
	_ = t.ImprovedConcussiveShot // Stun chance on Concussive Shot; Concussive Shot itself is not registered as a spell here, and a stun is CC, not damage.
	_ = t.ScatterShot            // A disorient that "turns off your attack when used" - a crowd-control tool, not a rotation shot.

	// Survival
	_ = t.Deflection       // Parry chance; defensive.
	_ = t.Entrapment       // Adds a snare/immobilize duration to trap triggers; a CC duration, not a damage number.
	_ = t.ImprovedWingClip // Chance to immobilize on Wing Clip; CC, and Wing Clip's own damage is unaffected.
	_ = t.Deterrence       // Dodge/Parry cooldown; Deterrence itself is not registered as a spell here.
	_ = t.SurvivalTactics  // Hit chance for Trap and Feign Death casts; neither is a damage roll.
}

func (hunter *Hunter) applyFrenzy() {
	if hunter.Talents.Frenzy == 0 {
		return
	}

	procChance := 0.2 * float64(hunter.Talents.Frenzy)

	procAura := hunter.pet.RegisterAura(core.Aura{
		Label:    "Frenzy Proc",
		ActionID: core.ActionID{SpellID: 19625},
		Duration: time.Second * 8,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1.3)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1/1.3)
		},
	})

	hunter.pet.RegisterAura(core.Aura{
		Label:    "Frenzy",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, spellResult *core.SpellResult) {
			if !spellResult.Outcome.Matches(core.OutcomeCrit) {
				return
			}
			if procChance == 1 || sim.RandomFloat("Frenzy") < procChance {
				procAura.Activate(sim)
			}
		},
	})
}

func (hunter *Hunter) registerBestialWrathCD() {
	if !hunter.Talents.BestialWrath {
		return
	}

	actionID := core.ActionID{SpellID: BestialWrathSpellId[0]}

	hunter.BestialWrathPetAura = hunter.pet.RegisterAura(core.Aura{
		Label:    "Bestial Wrath Pet",
		ActionID: actionID,
		Duration: time.Second * 18,
	}).AttachMultiplicativePseudoStatBuff(&hunter.pet.PseudoStats.DamageDealtMultiplier, 1.5)

	bwSpell := hunter.RegisterSpell(core.SpellConfig{
		ActionID:      actionID,
		Flags:         core.SpellFlagAPL,
		RequiredLevel: BestialWrathLevel[0],

		// spellconst/hunter.json's flat "cost" column reads 0 for
		// 19574 (BestialWrathManaCost[0]); Wowhead's Forever tooltip
		// gives Bestial Wrath as "12% of base mana", so this keeps the
		// percent-of-mana model (ManaCostOptions.BaseCost) rather than
		// reading the flat column as a real zero cost.
		ManaCost: core.ManaCostOptions{
			BaseCost: 0.12,
		},

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Millisecond * time.Duration(BestialWrathCooldownMS[0]),
			},
		},

		// RelatedSelfBuff is documented as "the aura this spell applies
		// to its caster" (sim/core/spell.go), but Bestial Wrath's own
		// 18s buff lives on the pet, not the hunter -- there is no
		// caster-side aura to point at instead. Wiring the pet aura in
		// here anyway is the only hook compare.go's engineDuration
		// reads without touching sim/core, and it reports the real
		// 18000ms duration instead of a false "missing" one.
		RelatedSelfBuff: hunter.BestialWrathPetAura,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			hunter.BestialWrathPetAura.Activate(sim)
		},
	})

	hunter.AddMajorCooldown(core.MajorCooldown{
		Spell: bwSpell,
		Type:  core.CooldownTypeDPS,
	})
}

// mortalShots is Mortal Shots' ranged crit damage bonus (19485, "all
// ranged abilities"). Its class mask holds the shots and Auto Shot and no
// melee strike, so only ranged spells read it.
func (hunter *Hunter) mortalShots() float64 {
	return 0.06 * float64(hunter.Talents.MortalShots)
}

// predatorsEdgeCritDamage is Predator's Edge's melee-critical-strike-damage
// half (client text: "Increases your melee critical strike damage by
// 6/12/18/24/30%", spell 1310627), the melee counterpart of Mortal Shots.
// Each melee special adds it to its CritDamageBonus directly, rather than
// through applyPredatorsEdge's OnSpellRegistered hook, so ranged shots
// stay out of it.
func (hunter *Hunter) predatorsEdgeCritDamage() float64 {
	return 0.06 * float64(hunter.Talents.PredatorsEdge)
}

// applyPredatorsEdge wires the other half of Predator's Edge -- the
// off-hand weapon damage bonus (10/20/30/40/50% per rank) -- onto every
// spell this hunter registers with an off-hand proc mask, including the
// off-hand auto attack itself (same OnSpellRegistered pattern as
// sim/warrior/talents.go's applyWeaponmaster).
func (hunter *Hunter) applyPredatorsEdge() {
	if hunter.Talents.PredatorsEdge == 0 {
		return
	}

	offHandMultiplier := 1 + 0.1*float64(hunter.Talents.PredatorsEdge)
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
			spell.DamageMultiplier *= offHandMultiplier
		}
	})
}

func (hunter *Hunter) applyTrapMastery() {
	/*
		if hunter.Talents.TrapMastery == 0 {
			return
		}

		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Flags.Matches(SpellFlagTrap) {
				spell.BonusHitRating += 5 * float64(hunter.Talents.TrapMastery)
			}
		})
	*/
}

func (hunter *Hunter) applyCleverTraps() {
	if hunter.Talents.CleverTraps == 0 {
		return
	}

	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagTrap) {
			spell.DamageMultiplier *= 1 + 0.15*float64(hunter.Talents.CleverTraps)
		}
	})
}

// efficiencyMeleeSpellCodes are the melee strikes in Efficiency's client
// class mask (19416): Raptor Strike, Mongoose Bite, Wing Clip,
// Counterattack and, since client 1.60.1.70707, Strider Kick (mask word 2
// went 0x2000 -> 0x202000; bit 21 is Strider Kick's 1317257 and the SoD
// Flanking Strike rune's ids, which no Forever hunter learns).
var efficiencyMeleeSpellCodes = []int32{
	SpellCode_HunterRaptorStrike,
	SpellCode_HunterMongooseBite,
	SpellCode_HunterWingClip,
	SpellCode_HunterCounterattack,
	SpellCode_HunterStriderKick,
}

func (hunter *Hunter) applyEfficiency() {
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		// Stings, Shots, Volley and the melee strikes in its class mask.
		// Parenthesised on purpose: `a && b || c` read Cost through a nil pointer whenever the
		// Volley clause matched a spell registered without a cost, which a character built
		// below 60 was the first to do.
		if spell.Cost != nil && (spell.Flags.Matches(SpellFlagSting|SpellFlagShot) || spell.SpellCode == SpellCode_HunterVolley || slices.Contains(efficiencyMeleeSpellCodes, spell.SpellCode)) {
			spell.Cost.Multiplier -= efficiencyCostReductionPerRank * hunter.Talents.Efficiency
		}
	})
}

// rankIndex clamps a talent rank to a lookup table's highest index -
// see sim/mage/talents.go's identical helper for why: core.FillTalentsProto
// does not validate a talent string against the client's max rank per
// node, so a corrupt or hand-edited string with more points in a talent
// than it allows would otherwise index one of the tables below out of
// range instead of reading the talent's own max-rank value.
func rankIndex[T any](rank int32, table []T) int {
	if i := int(rank); i >= 0 && i < len(table) {
		return i
	}
	return len(table) - 1
}

// applyDeadlyAspects is Deadly Aspects (talents/hunter.json node 104960,
// Beast Mastery, max rank 5, spell 19552): "While Aspect of the Hawk is
// active, Auto Shot has a 2/4/6/8/10% chance of increasing ranged attack
// speed by 30% for 12 sec. While Aspect of the Beast is active, all
// melee auto attacks have a [same]% chance of increasing melee attack
// speed by 30% for 12 sec."
//
// Its two chance mods (aura 107 misc 18) reach Aspect of the Hawk (effect 0,
// mask word0 1048576) and Aspect of the Beast (effect 1, mask word2 16)
// only. Aspect of the Falcon's mask is 0: its Quick Strikes proc is its own
// flat 5% (aspect_of_the_falcon.go), which the talent does not raise.
func (hunter *Hunter) applyDeadlyAspects() {
	if hunter.Talents.DeadlyAspects == 0 {
		return
	}

	procChance := 0.02 * float64(hunter.Talents.DeadlyAspects)
	speedAura := hunter.createImprovedHawkAura("Deadly Aspects Haste", core.ActionID{SpellID: 19552})
	quickStrikes := hunter.createQuickStrikesAura()
	var beast *core.Aura

	hunter.RegisterAura(core.Aura{
		Label:    "Deadly Aspects",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			beast = hunter.GetAura(aspectOfTheBeastAuraLabel)
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() {
				return
			}
			switch {
			case spell.ProcMask.Matches(core.ProcMaskRangedAuto) && hunter.HasActiveAuraWithTag(AspectOfTheHawkAuraTag):
				if sim.Proc(procChance, "Deadly Aspects") {
					speedAura.Activate(sim)
				}
			case spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) && beast != nil && beast.IsActive():
				// Beast's own Quick Strikes (1299448) is Falcon's 469144 again:
				// aura 319 at 30, 12 sec.
				if sim.Proc(procChance, "Deadly Aspects (Beast)") {
					quickStrikes.Activate(sim)
				}
			}
		},
	})
}

// applyFocusedFire is Focused Fire (talents/hunter.json node 104975,
// Beast Mastery, max rank 2, spell 1223755): "Increases all damage you
// and your pet deal by 1/2% while your pet is active."
//
// hunter.pet is only non-nil when a pet was actually brought
// (pet.go's NewHunterPet returns nil for Options.PetType PetNone or a
// zero PetUptime) - a build-time choice rather than a moment-to-moment
// one, the same approximation Lone Wolf's own "while you do not have an
// active pet" takes below in the opposite direction.
func (hunter *Hunter) applyFocusedFire() {
	if hunter.Talents.FocusedFire == 0 || hunter.pet == nil {
		return
	}

	mult := 1 + 0.01*float64(hunter.Talents.FocusedFire)
	hunter.PseudoStats.DamageDealtMultiplier *= mult
	hunter.pet.PseudoStats.DamageDealtMultiplier *= mult
}

// applyLethalAttacks is Lethal Attacks (talents/hunter.json node
// 105011, Marksmanship, max rank 5, spell 19426): "Increases your
// critical strike chance with all attacks by 1/2/3/4/5%." A flat Crit
// stat add, the same shape Surefooted's Hit add above uses, reaches
// every attack - melee, ranged, and spell alike - uniformly.
func (hunter *Hunter) applyLethalAttacks() {
	hunter.AddStat(stats.Crit, float64(hunter.Talents.LethalAttacks)*core.CritRatingPerCritChance)
}

// improvedStingsSerpentStingDamagePerRank is Improved Stings'
// Serpent-Sting-damage third (talents/hunter.json node 110870,
// Marksmanship, max rank 3, spell 1310661): "Increased the damage of
// your Serpent Sting ability by 6/13/20%." Not a clean per-rank
// multiple (6, +7, +7), so it is a table rather than a rate, the same
// convention sim/mage/talents.go's improvedConeOfColdDamage uses for
// the same reason.
var improvedStingsSerpentStingDamagePerRank = [4]float64{0, 0.06, 0.13, 0.20}

// applyImprovedStings wires the Serpent Sting damage third of Improved
// Stings onto serpent_sting.go's spell via OnSpellRegistered (the same
// pattern applyPredatorsEdge and applyCleverTraps use) rather than
// editing that file's own DamageMultiplier line, which already carries
// a different talent's bonus (Improved Serpent Sting, proto field
// improved_serpent_sting - not in this package's unread-field list).
//
// The other two-thirds of this talent's tooltip - "reduces the cooldown
// of your Viper Sting ability by 2/4/6 sec, and increases the duration
// of your Scorpid Sting ability by 15/30/45 sec" - have nothing to
// modify: neither Viper Sting nor Scorpid Sting is registered as a
// spell anywhere in this package (hunter.ScorpidSting is a declared
// field nothing ever assigns).
func (hunter *Hunter) applyImprovedStings() {
	if hunter.Talents.ImprovedStings == 0 {
		return
	}

	bonus := improvedStingsSerpentStingDamagePerRank[rankIndex(hunter.Talents.ImprovedStings, improvedStingsSerpentStingDamagePerRank[:])]
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_HunterSerpentSting {
			spell.DamageMultiplier *= 1 + bonus
		}
	})
}

// applyCarefulAim is Careful Aim (talents/hunter.json node 105008,
// Marksmanship, max rank 5, spell 1223984): "Increases your melee and
// ranged Attack Power by 20/40/60/80/100% of your Intellect." The client
// spell has both halves, aura 598 (melee) and aura 580 (ranged). Dynamic
// stat dependencies, so they stay correct when Intellect changes.
func (hunter *Hunter) applyCarefulAim() {
	if hunter.Talents.CarefulAim == 0 {
		return
	}

	share := 0.20 * float64(hunter.Talents.CarefulAim)
	hunter.AddStatDependency(stats.Intellect, stats.AttackPower, share)
	hunter.AddStatDependency(stats.Intellect, stats.RangedAttackPower, share)
}

// applyRapidKilling is Rapid Killing (talents/hunter.json node 105005,
// Marksmanship, max rank 2, spell 415405): "Reduces the cooldown on
// your Rapid Fire ability by 1/2 min. In addition, when you kill a
// non-trivial enemy or it dies while afflicted by your Serpent Sting,
// you gain Rapid Killing, increasing the damage of your next Shot
// ability within 20 sec by 10/20%."
//
// Only the Rapid Fire cooldown half is modeled. Rapid Fire itself is
// not registered until hunter.Initialize() (rapid_fire.go) runs, which
// is after ApplyTalents, so this reads it through OnSpellRegistered -
// keyed on its own spell id, 3045, the same ones every rank shares -
// the same way applyRangedWeaponSpecialization-style hooks elsewhere in
// this file reach a spell that does not exist yet.
//
// The kill-triggered damage buff is NOT modeled: it needs a target
// actually dying (or dying while Serpent Sting is ticking) mid-fight,
// and nothing in sim/core fires such an event - no OnUnitDeath, no
// OnTargetDies, nothing any class's talents.go listens to. A
// Patchwerk-style fixed-health-pool encounter here never marks a target
// dead at all. That is a sim/core gap this lane may not close.
func (hunter *Hunter) applyRapidKilling() {
	if hunter.Talents.RapidKilling == 0 {
		return
	}

	reduction := -time.Minute * time.Duration(hunter.Talents.RapidKilling)
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.ActionID.SpellID == 3045 {
			spell.CD.ApplyFlatCooldownMod(reduction)
		}
	})
}

// applyLoneWolf is Lone Wolf (talents/hunter.json node 105007,
// Marksmanship, bool, spell 415370): "You deal 20% increased damage
// with all attacks while you do not have an active pet."
//
// hunter.pet is nil only when Options.PetType is PetNone or PetUptime
// is zero (pet.go's NewHunterPet) - a build-time choice, not a
// moment-to-moment one, the same approximation applyFocusedFire takes
// above in the opposite direction.
func (hunter *Hunter) applyLoneWolf() {
	if !hunter.Talents.LoneWolf || hunter.pet != nil {
		return
	}

	hunter.PseudoStats.DamageDealtMultiplier *= 1.2
}

// rapidRecuperationSerpentStingRegen is Rapid Recuperation's
// Serpent-Sting-hit half (talents/hunter.json node 104999,
// Marksmanship, max rank 2, spell 1223987): "Hitting a target with your
// Serpent Sting ability grants you 25/50% ... of your Mana regeneration
// while casting for the next 15 sec."
var rapidRecuperationSerpentStingRegen = [3]float64{0, 0.25, 0.50}

// applyRapidRecuperation wires only the Serpent-Sting-hit half above.
// The tooltip's other half - "and consuming Rapid Killing grants you
// 50/100%" - is wired to the Rapid Killing kill-triggered proc buff
// applyRapidKilling documents as unmodeled (no target-death hook
// exists), so it can never fire here either.
func (hunter *Hunter) applyRapidRecuperation() {
	if hunter.Talents.RapidRecuperation == 0 {
		return
	}

	regen := rapidRecuperationSerpentStingRegen[rankIndex(hunter.Talents.RapidRecuperation, rapidRecuperationSerpentStingRegen[:])]
	regenAura := hunter.RegisterAura(core.Aura{
		Label:    "Rapid Recuperation",
		ActionID: core.ActionID{SpellID: 1223987},
		Duration: time.Second * 15,
	}).AttachAdditivePseudoStatBuff(&hunter.PseudoStats.SpiritRegenRateCasting, regen)

	hunter.RegisterAura(core.Aura{
		Label:    "Rapid Recuperation Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || spell.SpellCode != SpellCode_HunterSerpentSting {
				return
			}
			regenAura.Activate(sim)
		},
	})
}

// improvedTrackingMobTypes is every creature type Improved Tracking's
// tooltip names (talents/hunter.json node 104996, Survival, max rank 5,
// spell 24293): "While tracking Beasts, Demons, Dragonkin, Elementals,
// Giants, Humanoids, or Undead, all damage you deal to the tracked
// creature type is increased by 1/2/3/4/5%." (Mechanical and Critter are
// deliberately absent - the client's own list skips them too.)
var improvedTrackingMobTypes = []proto.MobType{
	proto.MobType_MobTypeBeast,
	proto.MobType_MobTypeDemon,
	proto.MobType_MobTypeDragonkin,
	proto.MobType_MobTypeElemental,
	proto.MobType_MobTypeGiant,
	proto.MobType_MobTypeHumanoid,
	proto.MobType_MobTypeUndead,
}

// improvedTrackingApplies says whether a hunter tracks this target. A
// target of no stated type (MobTypeUnknown) is the type-neutral boss the
// site ranks against, and every Phase 1 raid boss is a trackable type
// (Humanoid: Lucifron, Gehennas, Shazzrah, Sulfuron, Majordomo; Elemental:
// Garr, Geddon, Ragnaros; Giant: Golemagg; Beast: Magmadar; Dragonkin:
// Onyxia), so the hunter tracks it. A training dummy is no creature a
// Track spell finds.
func improvedTrackingApplies(mobType proto.MobType, dummy bool) bool {
	if mobType == proto.MobType_MobTypeUnknown {
		return !dummy
	}
	return slices.Contains(improvedTrackingMobTypes, mobType)
}

// applyImprovedTracking grants the damage bonus against every listed
// type unconditionally, rather than against only the one type a
// Track-spell selection would name: proto.Hunter_Options carries no
// Track-selection field at all in this package, so there is no way to
// read which single type the hunter is tracking. This is the same
// pool-not-prefix, AttackTables-mutating idiom
// sim/core/item_effects.go's NewMobTypeDamageEffect and
// sim/core/racials.go's Troll "Beast Slaying" use for an identical
// per-creature-type damage multiplier, deferred to
// RegisterPostFinalizeEffect because AttackTables is not allocated
// until Environment.setupAttackTables() runs, after ApplyTalents.
func (hunter *Hunter) applyImprovedTracking() {
	if hunter.Talents.ImprovedTracking == 0 {
		return
	}

	multiplier := 1 + 0.01*float64(hunter.Talents.ImprovedTracking)
	hunter.Env.RegisterPostFinalizeEffect(func() {
		for _, target := range hunter.Env.Encounter.AllTargetUnits {
			if !improvedTrackingApplies(target.MobType, hunter.Env.Encounter.Dummy) {
				continue
			}
			for _, at := range hunter.AttackTables[target.UnitIndex] {
				at.DamageDealtMultiplier *= multiplier
			}
		}
	})
}

// resourcefulnessProcChance is the critical-strike chance of the regen clause,
// 50% a rank (1.60.1.70425: "a 50%/100% chance"; the 1.60.1.70374 live text
// read 30/60%).
// bestialDisciplineRegenPerRank is Bestial Discipline's share of mana
// regeneration that continues while casting, a rank (19590: aura 134 at 50 for
// rank 2).
const bestialDisciplineRegenPerRank = 0.25

// resourcefulnessProcMask is the client's proc type mask for 440529
// (SpellAuraOptions 242512: 272, melee and ranged ability damage): a white
// swing's crit or a periodic tick's does not count.
const resourcefulnessProcMask = core.ProcMaskMeleeSpecial | core.ProcMaskRangedSpecial

func resourcefulnessProcChance(rank int32) float64 {
	return 0.5 * float64(rank)
}

// applyResourcefulness is Resourcefulness (talents/hunter.json node
// 104983, Survival, max rank 2, spell 440529): "Reduces the Mana cost of
// your Trap and melee abilities by 30/60%, and grants your melee and
// ranged ability Critical Strikes a 50/100% chance to allow 50% of your
// Mana Regeneration to continue while casting for 30 sec."
// (1.60.1.70425; the 1.60.1.70374 live text read 30/60% for the chance.)
//
// The cost cut (440529 effect 0, aura 108 misc 14) names its spells by
// class mask: 194 in word 0 (Raptor Strike, Mongoose Bite, Wing Clip, the
// traps), 524288 in word 1 (Counterattack), 2097152 in word 2 (Strider
// Kick). Every melee special but Lacerate is in it: Lacerate's 4096 in
// word 1 is in Savage Strikes' and Predator's Edge's masks (528384), not
// this one, so Lacerate keeps its full cost (1.60.1.70790 and 70888).
func (hunter *Hunter) applyResourcefulness() {
	if hunter.Talents.Resourcefulness == 0 {
		return
	}

	costReduction := -30 * hunter.Talents.Resourcefulness
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Cost == nil {
			return
		}
		if spell.SpellCode == SpellCode_HunterLacerate {
			return
		}
		if spell.Flags.Matches(SpellFlagTrap) || spell.ProcMask.Matches(core.ProcMaskMeleeMHSpecial|core.ProcMaskMeleeOHSpecial) {
			spell.Cost.Multiplier += costReduction
		}
	})

	procChance := resourcefulnessProcChance(hunter.Talents.Resourcefulness)
	regenAura := hunter.RegisterAura(core.Aura{
		Label:    "Resourcefulness",
		ActionID: core.ActionID{SpellID: 440529},
		Duration: time.Second * 30,
	}).AttachAdditivePseudoStatBuff(&hunter.PseudoStats.SpiritRegenRateCasting, 0.5)

	hunter.RegisterAura(core.Aura{
		Label:    "Resourcefulness Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.DidCrit() || !spell.ProcMask.Matches(resourcefulnessProcMask) {
				return
			}
			if sim.Proc(procChance, "Resourcefulness") {
				regenAura.Activate(sim)
			}
		},
	})
}

// exposePreyMongooseBiteProcChance is Expose Prey (talents/hunter.json
// node 104985, Survival, max rank 2, spell 1310532): "Your attacks
// against targets with Hunter's Mark have a 5/10% chance to activate
// your Mongoose Bite for 10 sec." (5 sec before build
// 1.60.1.70291)
var exposePreyMongooseBiteProcChance = [3]float64{0, 0.05, 0.10}

// exposePreyProcMask is the client's proc type mask for Expose Prey
// (SpellAuraOptions 340): melee and ranged auto attacks and specials.
const exposePreyProcMask = core.ProcMaskMeleeOrRanged

// applyExposePrey opens Mongoose Bite's activation window (the same
// Defensive State aura a dodge opens, held for 10 sec) on a landed melee or ranged
// attack against a Hunter's Mark target. The window aura stays nil
// until Initialize() registers Mongoose Bite (ApplyTalents runs first)
// and for a hunter below level 16, so it is read inside the proc closure.
func (hunter *Hunter) applyExposePrey() {
	if hunter.Talents.ExposePrey == 0 {
		return
	}

	procChance := exposePreyMongooseBiteProcChance[rankIndex(hunter.Talents.ExposePrey, exposePreyMongooseBiteProcChance[:])]

	hunter.RegisterAura(core.Aura{
		Label:    "Expose Prey",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if hunter.MongooseBiteWindowAura == nil || !result.Landed() || !spell.ProcMask.Matches(exposePreyProcMask) {
				return
			}
			if !result.Target.HasActiveAuraWithTag(core.HuntersMarkAuraTag) {
				return
			}
			if sim.Proc(procChance, "Expose Prey") {
				hunter.openMongooseBiteWindowFor(sim, exposePreyWindowLength)
			}
		},
	})
}

// applySurvivalistsDiscipline is Survivalist's Discipline
// (talents/hunter.json node 110860, Survival, max rank 2, spell
// 1310496): "Reduces the cooldown of your Trap and Deterrence abilities
// by 20/40%." Deterrence is not registered as a spell anywhere in this
// package, so only the Trap half has anything to reduce.
func (hunter *Hunter) applySurvivalistsDiscipline() {
	if hunter.Talents.SurvivalistsDiscipline == 0 {
		return
	}

	percent := -20 * int64(hunter.Talents.SurvivalistsDiscipline)
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagTrap) {
			spell.CD.ApplyFlatPercentCooldownMod(percent)
		}
	})
}
