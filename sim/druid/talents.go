package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
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

// clampRank clamps a talent rank to that talent's own max rank (the
// client's node data, data/builds/1.60.1.70009/talents/druid.json). This
// mirrors sim/mage/talents.go's identical rankOf/rankIndex helpers and
// the same root cause: core.FillTalentsProto does not validate a talent
// string against the client's max rank per node (sim/core/character.go),
// so a string with more points in a talent than the node allows (a
// hand-edited or stale one - this package's own balance_test.go
// P1Talents fixture over-ranks Nature's Majesty and Nature's Reach
// today) would otherwise read a bonus several multiples too large, or,
// for Eclipse's fixed-size lookup table below, panic.
func clampRank(rank int32, maxRank int32) int32 {
	if rank > maxRank {
		return maxRank
	}
	if rank < 0 {
		return 0
	}
	return rank
}

func (druid *Druid) ApplyTalents() {
	// Balance
	druid.registerMoonkinFormSpell()
	druid.applyOmenOfClarity()
	druid.applyImprovedMoonfire()
	druid.applyVengeance()
	druid.applyNaturesGrace()
	druid.applyMoonglow()
	druid.applyImprovedWrath()
	druid.applyMoonfury()
	druid.applyGenesis()
	druid.applyNaturesMajesty()
	druid.applyNaturesReach()
	druid.applyNaturesSplendor()
	druid.applyEclipse()
	druid.registerNaturesSwiftnessCD()
	druid.applyNaturalist()
	druid.applyLivingSpirit()

	/*
		druid.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= 1 + 0.02*float64(druid.Talents.NaturalWeapons)
	*/

	// Improved Entangling Roots (node 104926, 3 ranks) buffs a damage
	// and victim-damage-threshold effect on Entangling Roots, and
	// Overgrowth (node 110844, 2 ranks) raises its target cap.
	// Entangling Roots is a crowd-control spell this package does not
	// register, so neither changes a number here.
	_, _ = druid.Talents.ImprovedEntanglingRoots, druid.Talents.Overgrowth

	// Feral
	druid.applyBloodFrenzy()
	druid.applyPredatoryInstincts()
	druid.registerBerserkCD()

	druid.ApplyEquipScaling(stats.Armor, druid.ThickHideMultiplier())

	if druid.Talents.HeartOfTheWild > 0 {
		bonus := 0.04 * float64(druid.Talents.HeartOfTheWild)
		druid.MultiplyStat(stats.Intellect, 1.0+bonus)
	}

	// Feral Swiftness (node 104943): Cat Form movement speed and dodge
	// chance. Movement speed changes no cast or cost number, and this
	// DPS sim does not model incoming attacks dodging off the player
	// character, so dodge chance changes nothing here either.
	_ = druid.Talents.FeralSwiftness

	// Feral Instinct (node 104940): +10/20/30% Swipe damage and reduced
	// Prowl detection radius. Swipe is a Bear Form ability - Bear
	// Form's own damage kit is not modeled in this package (see
	// RegisterFeralCatSpells's comment above) - and detection radius
	// has no sim-side mechanic.
	_ = druid.Talents.FeralInstinct

	// Brutal Impact (node 104941): longer Bash/Pounce stuns and a
	// shorter Bash cooldown. Bash is a crowd-control ability this
	// package does not register.
	_ = druid.Talents.BrutalImpact

	// Feral Charge (node 104944): a Bear/Dire Bear Form gap closer and
	// interrupt. Bear Form's own damage kit is not modeled in this
	// package.
	_ = druid.Talents.FeralCharge

	// Primal Bite (proto field Mangle, node 104949 - the proto's own
	// field name is stale, see this file's header): a new Bear Form
	// finishing move, Rage-costed (spellconst/druid.json's cost_type 1
	// on spell 407995). Bear Form's own damage kit is not modeled in
	// this package, so registering it would be a spell no rotation can
	// ever reach - dead code this fork's conventions ask to delete
	// rather than add.
	_ = druid.Talents.Mangle

	// Natural Reaction (node 104954): Bear Form dodge chance and a
	// chance to gain Rage on dodge. Bear Form's own damage kit is not
	// modeled in this package.
	_ = druid.Talents.NaturalReaction

	// Restoration
	druid.applyFuror()

	druid.PseudoStats.SpiritRegenRateCasting += .05 * float64(druid.Talents.Reflection)

	// Nature's Focus (node 104957): a 14/28/42/56/70% chance to avoid
	// pushback on Arcane/Nature casts from taking damage. core/cast.go
	// models pushback, but (mirroring sim/mage/talents.go's identical
	// note on Improved Channeling) nothing in this sim's Patchwerk-style
	// fights interrupts a casting druid's own cast with incoming
	// damage, so avoiding it changes no number here.
	_ = druid.Talents.NaturesFocus

	// Subtlety (node 104920): -10/20/30% threat from Nature and Arcane
	// spells. Threat, not damage, mana or a cooldown.
	_ = druid.Talents.Subtlety

	// Gift of Nature, Gift of the Earthmother, Tranquil Spirit, Improved
	// Rejuvenation, Swiftmend, Improved Tranquility, Improved Regrowth
	// and Wild Growth are all healing-spell modifiers or a healing
	// finishing move. This package registers no healing spells -
	// Restoration is not brought up yet, the same state
	// RegisterFeralTankSpells's "TODO: Classic feral tank" comment
	// records for the Bear tank spec - so none of the eight has
	// anything to modify.
	_, _, _, _ = druid.Talents.GiftOfNature, druid.Talents.GiftOfTheEarthmother, druid.Talents.TranquilSpirit, druid.Talents.ImprovedRejuvenation
	_, _, _, _ = druid.Talents.Swiftmend, druid.Talents.ImprovedTranquility, druid.Talents.ImprovedRegrowth, druid.Talents.WildGrowth
}

func (druid *Druid) ThickHideMultiplier() float64 {
	thickHideMulti := 1.0

	if druid.Talents.ThickHide > 0 {
		thickHideMulti += 0.04 + 0.03*float64(druid.Talents.ThickHide-1)
	}

	return thickHideMulti
}

func (druid *Druid) BearArmorMultiplier() float64 {
	sotfMulti := 1.0 + 0.33/3.0
	return 4.7 * sotfMulti
}

// naturesGraceCastSpeed is Nature's Grace's proc (spell 16886): "Casting
// speed increased by 10% and global cooldown reduced by 10%" for 3 sec.
const (
	naturesGraceCastSpeed = 1.10
	naturesGraceDuration  = 3 * time.Second
)

// naturesGraceGlobalCooldownCut is 10% of the 1.5 s global cooldown.
const naturesGraceGlobalCooldownCut = 150 * time.Millisecond

// applyNaturesGrace implements Nature's Grace (node 104934, bool):
// "All non-periodic spell criticals grace you with a blessing of nature,
// increasing your spellcasting speed and reducing your global cooldown by
// 10% for 3 sec." The Classic version this replaced took a flat 0.5 s off
// the next cast's time for 15 s and ended on that cast.
func (druid *Druid) applyNaturesGrace() {
	if !druid.Talents.NaturesGrace {
		return
	}

	globalCooldownMod := druid.AddDynamicMod(core.SpellModConfig{
		Kind:      core.SpellMod_GlobalCooldown_Flat,
		ClassMask: DruidSpellMaskBalanceDamage,
		TimeValue: -naturesGraceGlobalCooldownCut,
	})

	druid.NaturesGraceProcAura = druid.RegisterAura(core.Aura{
		Label:    "Natures Grace Proc",
		ActionID: core.ActionID{SpellID: 16886},
		Duration: naturesGraceDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			druid.MultiplyCastSpeed(naturesGraceCastSpeed)
			globalCooldownMod.Activate()
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			druid.MultiplyCastSpeed(1 / naturesGraceCastSpeed)
			globalCooldownMod.Deactivate()
		},
	})

	core.MakePermanent(druid.RegisterAura(core.Aura{
		Label: "Natures Grace",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// Spells with travel times have their own implementation because the proc occurs as the cast finishes
			if spell.MissileSpeed == 0 && spell.ProcMask.Matches(core.ProcMaskSpellDamage) && result.DidCrit() {
				druid.NaturesGraceProcAura.Activate(sim)
			}
		},
	}))
}

// registerNaturesSwiftnessCD implements Nature's Swiftness (node 104921,
// bool, proto field NaturesSwiftness): "When activated, your next
// Nature spell becomes an instant cast spell." Wrath is this package's
// only Nature-school damage spell (Starfire is Arcane, Moonfire is
// Arcane), so only Wrath is affected - narrower than vanilla's version,
// which keyed off any spell school.
func (druid *Druid) registerNaturesSwiftnessCD() {
	if !druid.Talents.NaturesSwiftness {
		return
	}
	actionID := core.ActionID{SpellID: 17116}

	var affectedSpells []*DruidSpell
	var nsSpell *DruidSpell
	nsAura := druid.RegisterAura(core.Aura{
		Label:    "Natures Swiftness",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			affectedSpells = core.FilterSlice(druid.Wrath, func(ds *DruidSpell) bool { return ds != nil })
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.CastTimeMultiplier -= 1
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.CastTimeMultiplier += 1
			}
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.SpellCode != SpellCode_DruidWrath {
				return
			}

			// Remove the buff and put the cooldown back up.
			aura.Deactivate(sim)
			nsSpell.CD.Use(sim)
			druid.UpdateMajorCooldowns()
		},
	})

	nsSpell = druid.RegisterSpell(Humanoid|Moonkin, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,
		// Every client id for "Nature's Swiftness" (17116, 29274) is
		// spell_level 1; NatureSSwiftnessLevel[0] (constants_auto_gen.go)
		// reads the same.
		RequiredLevel: NatureSSwiftnessLevel[0],
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: time.Minute * 3,
			},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			nsAura.Activate(sim)
		},
	})
	druid.NaturesSwiftness = nsSpell

	druid.AddMajorCooldown(core.MajorCooldown{
		Spell: nsSpell.Spell,
		Type:  core.CooldownTypeDPS,
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			// Don't use NS unless we're casting a full-length Wrath.
			return !character.HasTemporarySpellCastSpeedIncrease()
		},
	})
}

// applyBloodFrenzy implements the proto's PrimalFury field, which is
// Blood Frenzy in 1.60.1.70009 (node 104947, tier 3 col 3 of the Feral
// tree; vanilla's Primal Fury moved to node 104947's neighbor, Shredding
// Attacks's column). Client text: "Gives you a 50/100% chance to gain an
// additional 5 Rage any time you get a critical strike while in Bear
// Form or Dire Bear Form. In addition, your non-periodic critical
// strikes from Cat Form abilities that generate Combo Points have a
// 50/100% chance to add an additional Combo Point." Only the Cat Form
// half is modeled: Bear Form's own damage kit (and so its Rage economy)
// is not modeled in this package.
func (druid *Druid) applyBloodFrenzy() {
	if druid.Talents.PrimalFury == 0 {
		return
	}

	rank := clampRank(druid.Talents.PrimalFury, 2)
	procChance := 0.5 * float64(rank)

	core.MakePermanent(druid.RegisterAura(core.Aura{
		Label: "Blood Frenzy",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !druid.InForm(Cat) ||
				!spell.Flags.Matches(SpellFlagBuilder) ||
				!result.Outcome.Matches(core.OutcomeCrit) {
				return
			}
			if sim.Proc(procChance, "Blood Frenzy") {
				druid.AddComboPoints(sim, 1, result.Target, spell.ComboPointMetrics())
			}
		},
	}))
}

// We're using an aura so that the APL can know if the Druid has furor for powershifting logic
func (druid *Druid) applyFuror() {
	if druid.Talents.Furor == 0 {
		return
	}

	spellID := []int32{0, 17056, 17058, 17059, 17060, 17061}[druid.Talents.Furor]

	druid.FurorAura = druid.RegisterAura(core.Aura{
		Label:    "Furor",
		ActionID: core.ActionID{SpellID: spellID},
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
	})
}

func (druid *Druid) applyOmenOfClarity() {
	/*
		if !druid.Talents.OmenOfClarity {
			return
		}

		var affectedSpells []*core.Spell
		druid.ClearcastingAura = druid.RegisterAura(core.Aura{
			Label:    "Clearcasting",
			ActionID: core.ActionID{SpellID: 16870},
			Duration: time.Second * 15,
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				affectedSpells = core.FilterSlice(druid.Spellbook, func(spell *core.Spell) bool { return spell.Flags.Matches(SpellFlagOmen) })
			},
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				for _, spell := range affectedSpells {
					spell.Cost.Multiplier -= 100
				}
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				for _, spell := range affectedSpells {
					spell.Cost.Multiplier += 100
				}
			},
			OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
				// OnCastComplete is called after OnSpellHitDealt / etc, so don't deactivate if it was just activated.
				if aura.RemainingDuration(sim) == aura.Duration {
					return
				}

				if spell.Flags.Matches(SpellFlagOmen) && spell.DefaultCast.Cost > 0 {
					aura.Deactivate(sim)
				}
			},
		})

		ppmm := druid.AutoAttacks.NewPPMManager(2.0, core.ProcMaskMelee)
		icd := core.Cooldown{
			Timer:    druid.NewTimer(),
			Duration: time.Second * 10,
		}

		druid.RegisterAura(core.Aura{
			Label:    "Omen of Clarity",
			Duration: core.NeverExpires,
			OnReset: func(aura *core.Aura, sim *core.Simulation) {
				aura.Activate(sim)
			},
			OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if !result.Landed() || !icd.IsReady(sim) {
					return
				}
				// TODO: Phase 3 "and non-instant spell casts" but we need to find out how the procs work for those
				if spell.ProcMask.Matches(core.ProcMaskMelee) && ppmm.ProcWithWeaponSpecials(sim, spell.ProcMask, "Omen of Clarity") {
					icd.Use(sim)
					druid.ClearcastingAura.Activate(sim)
				}
			},
		})
	*/
}

// moonfuryDamagePerRank is Moonfury's 2% a rank (node 104936, five ranks).
const moonfuryDamagePerRank = 0.02

// applyMoonfury implements Moonfury: "Increases the damage done by your
// Arcane and Nature spells by 2%" per rank. The client's spell (16896) is
// a school-damage aura, so it is a multiplier on both schools rather than
// an additive bonus on a list of spells: that list left out Insect Swarm,
// and made the bonus add with Improved Moonfire's instead of multiplying.
func (druid *Druid) applyMoonfury() {
	if druid.Talents.Moonfury == 0 {
		return
	}

	multiplier := 1 + moonfuryDamagePerRank*float64(clampRank(druid.Talents.Moonfury, 5))
	druid.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexArcane] *= multiplier
	druid.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexNature] *= multiplier
}

func (druid *Druid) applyImprovedMoonfire() {
	if druid.Talents.ImprovedMoonfire == 0 {
		return
	}

	// "Increases the damage and critical strike chance of your Moonfire
	// spell by 5%" per rank (the client's trait curve for spell 16821 reads
	// 5 and 10 on its crit, damage and periodic effects), where the Classic
	// tree this replaced gave 2% a rank.
	rank := float64(clampRank(druid.Talents.ImprovedMoonfire, 2))
	damageMultiplier := 0.05 * rank
	bonusCrit := 5 * rank * core.CritRatingPerCritChance

	druid.RegisterAura(core.Aura{
		Label: "Improved moonfire",
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			damageAffectedSpells := core.FilterSlice(
				druid.Moonfire,
				func(spell *DruidSpell) bool { return spell != nil },
			)

			critAffectedSpells := core.FilterSlice(
				druid.Moonfire,
				func(spell *DruidSpell) bool { return spell != nil },
			)

			for _, spell := range damageAffectedSpells {
				spell.BaseDamageMultiplierAdditive += damageMultiplier
			}

			for _, spell := range critAffectedSpells {
				spell.BonusCritRating += bonusCrit
			}
		},
	})
}

func (druid *Druid) applyVengeance() {
	if druid.Talents.Vengeance == 0 {
		return
	}

	critDamageBonus := 0.20 * float64(druid.Talents.Vengeance)

	druid.RegisterAura(core.Aura{
		Label: "Vengeance",
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			affectedSpells := core.FilterSlice(
				core.Flatten(
					[][]*DruidSpell{
						druid.Wrath,
						druid.Starfire,
						druid.Moonfire,
					},
				),
				func(spell *DruidSpell) bool { return spell != nil },
			)

			for _, spell := range affectedSpells {
				spell.CritDamageBonus += critDamageBonus
			}
		},
	})
}

// moonglowManaDiscount is Moonglow's 8/17/25% off "your damaging spells"
// (node 104925, three ranks).
var moonglowManaDiscount = [4]int64{0, 8, 17, 25}

// improvedWrathManaDiscountPerRank is Improved Wrath's 10% a rank off
// Wrath's mana cost; the 0.1 s a rank off its cast time is applied where
// Wrath is registered. The Classic port took only the cast time.
const improvedWrathManaDiscountPerRank = 10

// applyMoonglow implements Moonglow. The Classic port took 3 points a rank
// off Wrath, Starfire and Moonfire and took Starfire's off a second time
// in starfire.go; the client's class mask also names Insect Swarm.
func (druid *Druid) applyMoonglow() {
	if druid.Talents.Moonglow == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_PowerCost_Pct,
		ClassMask: DruidSpellMaskBalanceDamage,
		IntValue:  -moonglowManaDiscount[clampRank(druid.Talents.Moonglow, 3)],
	})
}

func (druid *Druid) applyImprovedWrath() {
	if druid.Talents.ImprovedWrath == 0 {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_PowerCost_Pct,
		ClassMask: DruidSpellMaskWrath,
		IntValue:  -improvedWrathManaDiscountPerRank * int64(clampRank(druid.Talents.ImprovedWrath, 5)),
	})
}

// applyNaturesMajesty implements Nature's Majesty (node 104927, 2
// ranks, proto field NaturesMajesty): "Increases your critical strike
// chance with spells and melee attacks by 2/4%." Forever's unified Crit
// stat (sim/core/stats/stats.go) already covers spell, melee and ranged
// crit together, so this is a flat stat addition rather than a
// per-spell SpellMod.
func (druid *Druid) applyNaturesMajesty() {
	if druid.Talents.NaturesMajesty == 0 {
		return
	}

	rank := clampRank(druid.Talents.NaturesMajesty, 2)
	druid.AddStat(stats.Crit, 2*float64(rank)*core.CritRatingPerCritChance)
}

// applyNaturesReach implements Nature's Reach (node 104929, 2 ranks,
// proto field NaturesReach): "Increases the range of your offensive
// Balance spells by 10/20% and improves your chance to hit by 2/4%."
// Range is not modeled (nothing in this sim ever goes out of range of
// its own target); hit is, as a SpellMod scoped to the three spells the
// client's text names - Wrath, Starfire and Moonfire.
func (druid *Druid) applyNaturesReach() {
	if druid.Talents.NaturesReach == 0 {
		return
	}

	rank := clampRank(druid.Talents.NaturesReach, 2)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusHit_Flat,
		ClassMask:  DruidSpellMaskBalanceDirectDamage,
		FloatValue: 2 * float64(rank) * core.HitRatingPerHitChance,
	})
}

// applyGenesis implements Genesis (node 104924, 5 ranks, proto field
// Genesis): "Increases the periodic damage and healing done by your
// spells and abilities by 1/2/3/4/5%." Only the damage half is modeled:
// this package registers no healing spells (see ApplyTalents's
// Restoration note). Scoped to every damage-over-time effect this
// package has - Moonfire, Insect Swarm, Rake and Rip.
func (druid *Druid) applyGenesis() {
	if druid.Talents.Genesis == 0 {
		return
	}

	rank := clampRank(druid.Talents.Genesis, 5)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_PeriodicDamageDone_Flat,
		ClassMask: DruidSpellMaskPeriodicDamage,
		IntValue:  int64(rank),
	})
}

// applyNaturesSplendor implements Nature's Splendor (node 104928, bool,
// proto field NaturesSplendor): "Increases the duration of your
// Moonfire and Rejuvenation spells by 3 sec, your Regrowth spell by 6
// sec, and your Insect Swarm spell by 2 sec." Rejuvenation and Regrowth
// are not modeled (no healing spells registered). Moonfire ticks every
// 3 sec and Insect Swarm every 2 sec, so +3/+2 sec is exactly +1 tick on
// each.
func (druid *Druid) applyNaturesSplendor() {
	if !druid.Talents.NaturesSplendor {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_DotNumberOfTicks_Flat,
		ClassMask: DruidSpellMaskMoonfire | DruidSpellMaskInsectSwarm,
		IntValue:  1,
	})
}

// applyEclipse implements Eclipse (node 104935, 3 ranks, proto field
// Eclipse): "Your Wrath spell reduces the cast time of your next 2
// Starfire spells by 0.17/0.33/0.5 sec. Stores up to 4 charges. Lasts
// 15 sec." Modeled as a 4-stack charge aura: each Wrath cast banks 2
// charges (clamped to 4 by Aura.SetStacks), each Starfire cast spends
// one, and the cast-time discount - which only ever has two values, "on"
// and "off", never scaling with stack count - is mutated directly on
// Starfire's DefaultCast.CastTime the moment the aura gains its first
// charge or loses its last, mirroring applyNaturesGrace's identical
// direct-mutation pattern above. A separate permanent watcher aura
// drives it, the same two-aura split applyMasterOfElements-equivalents
// elsewhere in this fork use, because OnCastComplete only fires for
// auras that are already active and this one is not active between
// charges.
func (druid *Druid) applyEclipse() {
	if druid.Talents.Eclipse == 0 {
		return
	}

	reduction := [4]time.Duration{0, 170 * time.Millisecond, 330 * time.Millisecond, 500 * time.Millisecond}[clampRank(druid.Talents.Eclipse, 3)]

	var affectedSpells []*DruidSpell
	eclipseAura := druid.RegisterAura(core.Aura{
		Label:     "Eclipse",
		ActionID:  core.ActionID{SpellID: 408248},
		Duration:  time.Second * 15,
		MaxStacks: 4,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			affectedSpells = core.FilterSlice(druid.Starfire, func(ds *DruidSpell) bool { return ds != nil })
		},
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
			if oldStacks == 0 && newStacks > 0 {
				for _, spell := range affectedSpells {
					spell.DefaultCast.CastTime -= reduction
				}
			} else if oldStacks > 0 && newStacks == 0 {
				for _, spell := range affectedSpells {
					spell.DefaultCast.CastTime += reduction
				}
			}
		},
	})

	core.MakePermanent(druid.RegisterAura(core.Aura{
		Label: "Eclipse Trigger",
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			switch spell.SpellCode {
			case SpellCode_DruidWrath:
				eclipseAura.Activate(sim)
				eclipseAura.AddStacks(sim, 2)
			case SpellCode_DruidStarfire:
				if eclipseAura.IsActive() {
					eclipseAura.RemoveStack(sim)
				}
			}
		},
	}))
}

// applyPredatoryInstincts implements Predatory Instincts (node 104950,
// 2 ranks, proto field PredatoryInstincts): "Increases the critical
// strike damage bonus of your melee abilities by 10/20%." Scoped to
// every Cat Form melee special this package registers.
func (druid *Druid) applyPredatoryInstincts() {
	if druid.Talents.PredatoryInstincts == 0 {
		return
	}

	rank := clampRank(druid.Talents.PredatoryInstincts, 2)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_CritDamageBonus_Flat,
		ClassMask:  DruidSpellMaskMeleeAbilities,
		FloatValue: 0.10 * float64(rank),
	})
}

// registerBerserkCD implements Berserk (node 104956, bool, proto field
// Berserk): "Causes your Primal Bite ability to strike up to 3 targets,
// removes its cooldown, and increases the critical strike chance of
// your Combo Point-generating abilities by 100%. Clears and grants
// immunity to Fear effects for the duration. Lasts 15 sec." Only the
// Combo-Point-generator crit bonus is modeled: Primal Bite is a Bear
// Form ability and Bear Form's own damage kit is not modeled in this
// package (see ApplyTalents's Mangle note), and Fear immunity has no
// mechanic on this sim's fights.
func (druid *Druid) registerBerserkCD() {
	if !druid.Talents.Berserk {
		return
	}

	actionID := core.ActionID{SpellID: 417141}
	bonusCrit := 100.0 * core.CritRatingPerCritChance

	var affectedSpells []*DruidSpell
	berserkAura := druid.RegisterAura(core.Aura{
		Label:    "Berserk",
		ActionID: actionID,
		Duration: time.Second * 15,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			affectedSpells = core.FilterSlice(
				[]*DruidSpell{druid.Claw, druid.Rake, druid.Ravage, druid.Shred},
				func(ds *DruidSpell) bool { return ds != nil },
			)
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.BonusCritRating += bonusCrit
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.BonusCritRating -= bonusCrit
			}
		},
	})

	druid.Berserk = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,
		// Every client id for "Berserk" (417141, 424759, 442211) is
		// spell_level 1; BerserkLevel[0] (constants_auto_gen.go) reads
		// the same.
		RequiredLevel: BerserkLevel[0],
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: time.Minute * 3,
			},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			berserkAura.Activate(sim)
		},
	})

	druid.AddMajorCooldown(core.MajorCooldown{
		Spell: druid.Berserk.Spell,
		Type:  core.CooldownTypeDPS,
	})
}

// applyNaturalist implements Naturalist (node 104922, 5 ranks, proto
// field Naturalist): "Reduces the cast time of your Healing Touch spell
// by 0.1/.../0.5 sec and increases all damage you deal by 1/2/3/4/5%."
// Healing Touch is not modeled (no healing spells registered); the
// damage half applies to literally everything the character does, so
// it is a flat PseudoStats multiplier rather than a per-spell mask.
func (druid *Druid) applyNaturalist() {
	if druid.Talents.Naturalist == 0 {
		return
	}

	rank := clampRank(druid.Talents.Naturalist, 5)
	druid.PseudoStats.DamageDealtMultiplier *= 1 + 0.01*float64(rank)
}

// applyLivingSpirit implements Living Spirit (node 104911, 3 ranks,
// proto field LivingSpirit): "Increases your Spirit by 5/10/15%."
func (druid *Druid) applyLivingSpirit() {
	if druid.Talents.LivingSpirit == 0 {
		return
	}

	rank := clampRank(druid.Talents.LivingSpirit, 3)
	druid.MultiplyStat(stats.Spirit, 1+0.05*float64(rank))
}
