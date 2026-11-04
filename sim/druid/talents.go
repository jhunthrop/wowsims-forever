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
	druid.applyNaturalInstinct()

	druid.applyHeartOfTheWildIntellect()

	druid.applyFeralDodge()
	druid.applyNaturalReaction()

	// Feral Instinct (swipe.go), Brutal Impact (Bash stun duration and
	// cooldown: a crowd-control ability the tank sim has no fight use
	// for), Feral Charge (a gap closer), Shredding Attacks, Ferocity and
	// Savage Fury (read by the abilities they discount) are accounted for
	// in the ability files.
	_, _ = druid.Talents.BrutalImpact, druid.Talents.FeralCharge

	// Restoration
	druid.applyFuror()

	druid.applyReflection()

	druid.applyNaturesFocus()

	// Subtlety (node 104917): -10/20/30% threat from Nature and Arcane
	// spells. Threat, not damage, mana or a cooldown.
	_ = druid.Talents.Subtlety

	druid.applyGiftOfNature()
	druid.applyTranquilSpirit()
	druid.applyImprovedRejuvenation()
	druid.applyImprovedTranquility()
	druid.applyImprovedRegrowth()
	// Swiftmend and Wild Growth are the spells their talents grant; they
	// register in RegisterHealingSpells (swiftmend.go, wild_growth.go).
}

// naturesSplendorRegrowthTicks is Nature's Splendor's 6 sec on Regrowth in
// its 3 sec ticks.
const naturesSplendorRegrowthTicks = 2

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
//
// "Spell criticals" includes heals: 16880's ProcTypeMask 0x14000 takes
// helpful magic spells as well as harmful ones, and the proc's global
// cooldown cut (16886 effect 1, mask 0x7407f7 / 0x4000000) names Healing
// Touch, Regrowth, Rejuvenation, Tranquility and Wild Growth beside the
// Balance spells. A tick is periodic, so it never procs.
func (druid *Druid) applyNaturesGrace() {
	if !druid.Talents.NaturesGrace {
		return
	}

	globalCooldownMod := druid.AddDynamicMod(core.SpellModConfig{
		Kind:      core.SpellMod_GlobalCooldown_Flat,
		ClassMask: DruidSpellMaskBalanceDamage | DruidSpellMaskNaturesGraceHeals,
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
			// Spells with travel times (Wrath) activate it in their own landing callback.
			if spell.MissileSpeed == 0 && spell.ProcMask.Matches(core.ProcMaskSpellDamage) && result.DidCrit() {
				druid.NaturesGraceProcAura.Activate(sim)
			}
		},
		OnHealDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.ProcMask.Matches(core.ProcMaskSpellHealing) && result.DidCrit() {
				druid.NaturesGraceProcAura.Activate(sim)
			}
		},
	}))
}

// registerNaturesSwiftnessCD implements Nature's Swiftness (node 104921,
// bool, proto field NaturesSwiftness): "When activated, your next
// Nature spell becomes an instant cast spell." The Nature spells with a
// cast time are Wrath (Starfire and Moonfire are Arcane) and the three
// cast-time heals, Healing Touch, Regrowth and Wild Growth -
// DruidSpellMaskNatureCasts.
func (druid *Druid) registerNaturesSwiftnessCD() {
	if !druid.Talents.NaturesSwiftness {
		return
	}
	actionID := core.ActionID{SpellID: 17116}

	var affectedSpells []*core.Spell
	var nsSpell *DruidSpell
	nsAura := druid.RegisterAura(core.Aura{
		Label:    "Natures Swiftness",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			affectedSpells = core.FilterSlice(druid.Spellbook, func(spell *core.Spell) bool {
				return spell.Matches(DruidSpellMaskNatureCasts)
			})
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
			if !spell.Matches(DruidSpellMaskNatureCasts) {
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
			// A healer's rotation casts it by hand, ahead of the emergency
			// heal it makes instant.
			if druid.CastsNaturesSwiftnessByHand {
				return false
			}
			// Don't use NS unless we're casting a full-length Wrath.
			return !character.HasTemporarySpellCastSpeedIncrease()
		},
	})
}

// applyBloodFrenzy implements Blood Frenzy (node 104947, proto field
// BloodFrenzy). Client text: "Gives you a 50/100% chance to gain an
// additional 5 Rage any time you get a critical strike while in Bear
// Form or Dire Bear Form. In addition, your non-periodic critical
// strikes from Cat Form abilities that generate Combo Points have a
// 50/100% chance to add an additional Combo Point." A bear's melee crit
// (auto-attack or ability) pays the rage; a cat's builder crit pays the
// combo point.
func (druid *Druid) applyBloodFrenzy() {
	if druid.Talents.BloodFrenzy == 0 {
		return
	}

	rank := clampRank(druid.Talents.BloodFrenzy, 2)
	procChance := 0.5 * float64(rank)
	rageMetrics := druid.NewRageMetrics(core.ActionID{SpellID: 16958})

	core.MakePermanent(druid.RegisterAura(core.Aura{
		Label: "Blood Frenzy",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Outcome.Matches(core.OutcomeCrit) {
				return
			}
			switch {
			case druid.InForm(Bear) && spell.ProcMask.Matches(core.ProcMaskMelee):
				if sim.Proc(procChance, "Blood Frenzy") {
					druid.AddRage(sim, bloodFrenzyRage, rageMetrics)
				}
			case druid.InForm(Cat) && spell.Flags.Matches(SpellFlagBuilder):
				if sim.Proc(procChance, "Blood Frenzy") {
					druid.AddComboPoints(sim, 1, result.Target, spell.ComboPointMetrics())
				}
			}
		},
	}))
}

// bloodFrenzyRage is the "additional 5 Rage" of a bear's critical strike.
const bloodFrenzyRage = 5.0

// We're using an aura so that the APL can know if the Druid has furor for powershifting logic
// furorSpellID is Furor's one talent spell at every rank in 1.60.1.70707.
const furorSpellID int32 = 17056

func (druid *Druid) applyFuror() {
	if druid.Talents.Furor == 0 {
		return
	}

	spellID := furorSpellID

	druid.FurorAura = druid.RegisterAura(core.Aura{
		Label:    "Furor",
		ActionID: core.ActionID{SpellID: spellID},
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
	})
}

// Omen of Clarity (spell 16864) is a baseline trainer spell in Forever
// (SkillLineAbility: skill line 574, AcquireMethod 0, level 20), not the
// vanilla Restoration talent. Client text: "Your spells and attacks have a
// chance to grant you Clearcasting, reducing the Mana, Rage, or Energy cost
// of your next damage or healing spell or offensive ability by 100%.
// Clearcasting is not consumed by Wrath or by spells or abilities that cost
// no resources." Its proc row (SpellAuraOptions) is ProcChance 100 with a
// 10 s ProcCategoryRecovery on melee hits, harmful spells and heals, but no
// periodic damage; the client's 100% is not the server's rate, which the
// row does not carry.
//
// The rate: melee keeps the engine's 2 procs a minute, yellow hits at the
// equipped weapon's speed (the Classic and Season of Discovery model this
// fork inherited); beta cat logs agree (10 cats on foreverlogs.gg, 346
// Clearcasting gains against about 330 at 2 a minute on the 1.0 s paw). A
// spell has a flat 4% a cast, whatever its cast time: 34 caster druids'
// beta logs (12,402 casts, 547 gains) show no rise for cast-time spells
// over instants and about 4% a spell once melee procs are taken out, where
// 2 a minute over the cast time predicted about 840; the one stated number
// (Reddit, 2026-09-30) is "4% chance on spells and 10 sec internal cd".
const (
	omenOfClarityPPM             = 2.0
	omenOfClaritySpellProcChance = 0.04
	omenOfClarityCooldown        = 10 * time.Second // SpellAuraOptions.ProcCategoryRecovery
	clearcastingDuration         = 15 * time.Second // spell 16870, DurationIndex 8

	// Moonkin Form (spell 24858) effect 4 adds 100% to Omen of Clarity's
	// chance (SpellModOp 18, proc chance) and effect 5 takes 50% off its
	// proc cooldown (SpellModOp 38).
	moonkinOmenChanceMultiplier   = 2.0
	moonkinOmenCooldownMultiplier = 0.5
)

// omenSpellProcChance is a spell's chance to grant Clearcasting, the same
// for every cast time.
func omenSpellProcChance(inMoonkin bool) float64 {
	chance := omenOfClaritySpellProcChance
	if inMoonkin {
		chance *= moonkinOmenChanceMultiplier
	}
	return chance
}

// omenProcCooldown is Omen of Clarity's proc cooldown.
func omenProcCooldown(inMoonkin bool) time.Duration {
	if inMoonkin {
		return time.Duration(float64(omenOfClarityCooldown) * moonkinOmenCooldownMultiplier)
	}
	return omenOfClarityCooldown
}

// ProcClearcasting is Omen of Clarity's proc from spell: one charge of
// Clearcasting (16870).
func (druid *Druid) ProcClearcasting(sim *core.Simulation, spell *core.Spell) {
	druid.clearcastingProc = core.ProcSource{Spell: spell, At: sim.CurrentTime}
	druid.ClearcastingAura.Activate(sim)
}

func (druid *Druid) applyOmenOfClarity() {
	if int(druid.Level) < OmenOfClarityLevel[0] {
		return
	}

	// Clearcasting's own class mask (spell 16870) holds every druid damage,
	// heal and feral ability that carries SpellFlagOmen; Wrath is not in it.
	freeCost := druid.AddDynamicMod(core.SpellModConfig{
		Kind:       core.SpellMod_PowerCost_Pct,
		SpellFlags: SpellFlagOmen,
		IntValue:   -100,
	})
	druid.ClearcastingAura = druid.RegisterAura(core.Aura{
		Label:    "Clearcasting",
		ActionID: core.ActionID{SpellID: ClearcastingSpellId[0]},
		Duration: clearcastingDuration,
		OnGain: func(_ *core.Aura, _ *core.Simulation) {
			freeCost.Activate()
		},
		OnExpire: func(_ *core.Aura, _ *core.Simulation) {
			freeCost.Deactivate()
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			// The cast that procced the charge does not spend it. Any
			// other cast does, one completing at the proc's own moment
			// included (a Shred right after the swing that procced).
			if druid.clearcastingProc.IsFrom(sim, spell) {
				return
			}
			druid.spendClearcasting(sim, spell)
		},
	})

	ppmm := druid.AutoAttacks.NewPPMManager(omenOfClarityPPM, core.ProcMaskMelee)
	var readyAt time.Duration

	roll := func(sim *core.Simulation, spell *core.Spell) {
		if sim.CurrentTime < readyAt {
			return
		}
		inMoonkin := druid.InForm(Moonkin)
		var procced bool
		if spell.ProcMask.Matches(core.ProcMaskMelee) {
			procced = ppmm.ProcWithWeaponSpecials(sim, spell.ProcMask, "Omen of Clarity")
		} else if spell.ProcMask.Matches(core.ProcMaskSpellDamage | core.ProcMaskSpellHealing) {
			// A rank far below the druid's level rolls at its level-gap share
			// (Blizzard's beta note names Omen of Clarity; core.Spell.LevelPenaltyShare).
			procced = sim.Proc(omenSpellProcChance(inMoonkin)*spell.LevelPenaltyShare(), "Omen of Clarity")
		}
		if procced {
			readyAt = sim.CurrentTime + omenProcCooldown(inMoonkin)
			druid.ProcClearcasting(sim, spell)
		}
	}
	tryProc := func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
		if result.Landed() {
			roll(sim, spell)
		}
	}

	core.MakePermanent(druid.RegisterAura(core.Aura{
		Label:    "Omen of Clarity",
		ActionID: core.ActionID{SpellID: OmenOfClaritySpellId[0]},
		OnReset: func(_ *core.Aura, _ *core.Simulation) {
			readyAt = 0
		},
		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			tryProc(sim, spell, result)
		},
		OnHealDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			tryProc(sim, spell, result)
		},
		// A heal over time with no direct heal (Rejuvenation, Tranquility,
		// Wild Growth) never reaches OnHealDealt, but casting it is a
		// helpful spell landing, which the proc row's 0x4000 bit names:
		// it rolls once a cast.
		OnCastComplete: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.ClassSpellMask&DruidSpellMaskHotOnlyHeals != 0 {
				roll(sim, spell)
			}
		},
	}))
}

// DruidSpellMaskHotOnlyHeals is the heals that land only as a heal over
// time, with no direct heal of their own.
const DruidSpellMaskHotOnlyHeals = DruidSpellMaskRejuvenation | DruidSpellMaskTranquility | DruidSpellMaskWildGrowth

// spendClearcasting spends Clearcasting's one charge (16870, ProcCharges 1)
// on an ability in its class mask that has a cost. OnCastComplete calls it;
// so must an ability cast with SpellFlagNoOnCastComplete (Maul, off the
// swing), before it lands, or its own proc would be the one spent.
func (druid *Druid) spendClearcasting(sim *core.Simulation, spell *core.Spell) {
	if druid.ClearcastingAura.IsActive() && spell.Flags.Matches(SpellFlagOmen) && spell.DefaultCast.Cost > 0 {
		druid.ClearcastingAura.Deactivate(sim)
	}
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
						// Client 16909's mask 0x600207 holds Insect
						// Swarm (0x200000) as well.
						druid.InsectSwarm,
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
// Range is not modeled (nothing in this sim ever goes out of range of its
// own target). The hit is client 16819's effects 1 and 2, aura 54 (melee
// hit) and aura 55 (spell hit), neither with a class mask: hit on every
// spell and attack, Insect Swarm included. Only the range effect names
// spells.
func (druid *Druid) applyNaturesReach() {
	if druid.Talents.NaturesReach == 0 {
		return
	}

	rank := clampRank(druid.Talents.NaturesReach, 2)
	druid.AddStat(stats.Hit, 2*float64(rank)*core.HitRatingPerHitChance)
}

// genesisPeriodicPerRank is Genesis's 2% a rank. The client row (spell
// 1223081, aura 108 points 1) and its pre-hotfix text say 1%; client
// 1.60.1.70707's hotfixed text says 2/4/6/8/10%.
const genesisPeriodicPerRank = 2

// applyGenesis implements Genesis (node 104924, 5 ranks, proto field
// Genesis): "Increases the periodic damage and healing done by your
// spells and abilities by 2/4/6/8/10%." Scoped to every damage-over-time
// and heal-over-time effect this package has - Moonfire, Insect Swarm, Rake
// and Rip, Rejuvenation, Regrowth's heal over time and Wild Growth; the
// client's mask leaves Tranquility out.
func (druid *Druid) applyGenesis() {
	if druid.Talents.Genesis == 0 {
		return
	}

	rank := clampRank(druid.Talents.Genesis, 5)
	druid.AddStaticMod(core.SpellModConfig{
		Kind: core.SpellMod_PeriodicDamageDone_Flat,
		// Client 1223081's mask names Rejuvenation, Regrowth and Wild
		// Growth among the heals, not Tranquility (0x80).
		ClassMask: DruidSpellMaskPeriodicDamage | (DruidSpellMaskPeriodicHealing &^ DruidSpellMaskTranquility),
		IntValue:  genesisPeriodicPerRank * int64(rank),
	})
}

// applyNaturesSplendor implements Nature's Splendor (node 104928, bool,
// proto field NaturesSplendor): "Increases the duration of your
// Moonfire and Rejuvenation spells by 3 sec, your Regrowth spell by 6
// sec, and your Insect Swarm spell by 2 sec." Moonfire and Rejuvenation
// tick every 3 sec, Regrowth every 3 sec and Insect Swarm every 2 sec, so
// the extra time is one tick on Moonfire, Rejuvenation and Insect Swarm
// and two on Regrowth.
func (druid *Druid) applyNaturesSplendor() {
	if !druid.Talents.NaturesSplendor {
		return
	}

	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_DotNumberOfTicks_Flat,
		ClassMask: DruidSpellMaskMoonfire | DruidSpellMaskInsectSwarm | DruidSpellMaskRejuvenation,
		IntValue:  1,
	})
	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_DotNumberOfTicks_Flat,
		ClassMask: DruidSpellMaskRegrowth,
		IntValue:  naturesSplendorRegrowthTicks,
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

// naturalInstinctHealingFromIntellect is the healing clause by rank:
// "increases your spell healing by 12%/25% of your Intellect" (not a
// clean multiple, so a table).
var naturalInstinctHealingFromIntellect = [3]float64{0, 0.12, 0.25}

// applyNaturalInstinct implements Natural Instinct (node 104950, 2
// ranks, Predatory Instincts' slot since 1.60.1.70425): "Increases the
// critical strike damage bonus of your melee abilities by 10/20% and
// increases your spell healing by 12/25% of your Intellect." The crit
// half is scoped to every Cat Form melee special this package registers;
// the healing half is a stat dependency, so a Restoration druid taking
// the talent reads it on every heal.
func (druid *Druid) applyNaturalInstinct() {
	if druid.Talents.NaturalInstinct == 0 {
		return
	}

	rank := clampRank(druid.Talents.NaturalInstinct, 2)
	druid.AddStaticMod(core.SpellModConfig{
		Kind: core.SpellMod_CritDamageBonus_Flat,
		// Client 1223242's mask 0x839800 holds Rip (0x800000) besides the
		// melee specials.
		ClassMask:  DruidSpellMaskMeleeAbilities | DruidSpellMaskRip,
		FloatValue: 0.10 * float64(rank),
	})
	druid.AddStatDependency(stats.Intellect, stats.HealingPower, naturalInstinctHealingFromIntellect[rank])
}

// naturalistCastTimeCut is Naturalist's 0.1 sec a rank off Healing Touch
// and Wild Growth, and naturalistDamagePerRank its 2% a rank to all damage
// (1% and Healing Touch alone before client 1.60.1.70707's hotfix).
const (
	naturalistCastTimeCut   = 100 * time.Millisecond
	naturalistDamagePerRank = 0.02
)

// applyNaturalist implements Naturalist (node 104922, 5 ranks, proto
// field Naturalist): "Reduces the cast time of your Healing Touch and Wild
// Growth spells by 0.1/.../0.5 sec and increases all damage you deal by
// 2/4/6/8/10%."
// The damage half applies to literally everything the character does,
// so it is a flat PseudoStats multiplier rather than a per-spell mask.
func (druid *Druid) applyNaturalist() {
	if druid.Talents.Naturalist == 0 {
		return
	}

	rank := clampRank(druid.Talents.Naturalist, 5)
	druid.PseudoStats.DamageDealtMultiplier *= 1 + naturalistDamagePerRank*float64(rank)
	druid.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_CastTime_Flat,
		ClassMask: DruidSpellMaskHealingTouch | DruidSpellMaskWildGrowth,
		TimeValue: -naturalistCastTimeCut * time.Duration(rank),
	})
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

// naturesFocusPushbackReductionPerRank is Nature's Focus (node 104957):
// "Gives your Arcane and Nature spells a 14/28/42/56/70% chance to not
// lose casting time when you take damage." The healer takes raid-wide
// pulses mid-cast, and core's pushback reads this chance off the spell
// being cast.
const naturesFocusPushbackReductionPerRank = 0.14

// DruidSpellMaskNaturesFocus is what the talent's pushback modifier names
// (17063 effect 0, aura 108 misc 9), not its "Arcane and Nature spells"
// text: Healing Touch, Regrowth, Tranquility, Wrath and Starfire among the
// spells registered here (Hurricane, Entangling Roots and Rebirth are not
// registered). Wild Growth, whose 3 s cast the server's push added, is
// not in the mask.
const DruidSpellMaskNaturesFocus = DruidSpellMaskHealingTouch | DruidSpellMaskRegrowth |
	DruidSpellMaskTranquility | DruidSpellMaskWrath | DruidSpellMaskStarfire

func (druid *Druid) applyNaturesFocus() {
	if druid.Talents.NaturesFocus == 0 {
		return
	}
	reduction := naturesFocusPushbackReductionPerRank * float64(druid.Talents.NaturesFocus)
	druid.OnSpellRegistered(func(spell *core.Spell) {
		if spell.ClassSpellMask&DruidSpellMaskNaturesFocus != 0 {
			spell.PushbackReduction += reduction
		}
	})
}
