package paladin

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

func (paladin *Paladin) ApplyTalents() {
	// TODO: paladin.AddStat(stats.RangedHit, float64(paladin.Talents.Precision)*core.HitRatingPerHitChance)

	paladin.AddStat(stats.Crit, float64(paladin.Talents.Conviction)*core.CritRatingPerCritChance)
	// TODO: paladin.AddStat(stats.RangedCrit, float64(paladin.Talents.Conviction)*core.CritRatingPerCritChance)

	// These are no-op if untalented.
	paladin.MultiplyStat(stats.Strength, 1.0+0.02*float64(paladin.Talents.DivineStrength))
	paladin.MultiplyStat(stats.Intellect, 1.0+0.02*float64(paladin.Talents.DivineIntellect))

	paladin.AddStat(stats.Parry, 1*float64(paladin.Talents.Deflection))

	paladin.applyWeaponSpecialization()
	if paladin.Talents.Vengeance > 0 {
		paladin.applyVengeance()
	}
	if paladin.Talents.Vindication > 0 {
		paladin.applyVindication()
	}
	paladin.PseudoStats.SchoolBonusCritChance[stats.SchoolIndexHoly] += core.CritRatingPerCritChance * float64(paladin.Talents.HolyPower)

	paladin.applyProtectionTalents()
	paladin.applyImprovedLayOnHands()

	// Client trait-tree talents (talents/paladin.json, build
	// 1.60.1.70009). Pure spell modifiers are declarative config, below;
	// anything with state, a timer or a proc keeps its own function.
	paladin.applyDeclarativeTalents()
	paladin.applySanctifiedJudgement()
	paladin.registerTwistOfLight()
	paladin.markUnmodeledForeverTalents()
}

func (paladin *Paladin) improvedSoR() float64 {
	// FOREVER: Improved Seal of Righteousness is not in the client's trees.
	// return []float64{1, 1.03, 1.06, 1.09, 1.12, 1.15}[paladin.Talents.ImprovedSealOfRighteousness]
	return 1
}

func (paladin *Paladin) benediction() int32 {
	return []int32{100, 97, 94, 91, 88, 85}[paladin.Talents.Benediction]
}

// getWeaponSpecializationModifier is the damage multiplier of the weapon
// in the main hand. One-Handed Weapon Specialization is "Increases the
// damage you deal with one-handed melee weapons by 3%/7%/10%"
// (oneHandedWeaponSpecializationPct); Two-Handed Weapon Specialization is
// 2% a rank.
func (paladin *Paladin) getWeaponSpecializationModifier() float64 {
	handType := paladin.MainHand().HandType
	if handType == proto.HandType_HandTypeMainHand || handType == proto.HandType_HandTypeOneHand {
		return 1. + oneHandedWeaponSpecializationPct[paladin.Talents.OneHandedWeaponSpecialization]
	} else if handType == proto.HandType_HandTypeTwoHand {
		return 1. + 0.02*float64(paladin.Talents.TwoHandedWeaponSpecialization)
	} else {
		return 1.
	}
}

// Affects all physical damage or spells that can be rolled as physical.
func (paladin *Paladin) applyWeaponSpecialization() {
	paladin.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= paladin.getWeaponSpecializationModifier()
}

func (paladin *Paladin) applyVengeance() {
	if paladin.Talents.Vengeance == 0 {
		return
	}

	vengeanceMultiplier := []float64{1, 1.03, 1.06, 1.09, 1.12, 1.15}[paladin.Talents.Vengeance]

	procAura := paladin.RegisterAura(core.Aura{
		Label:    "Vengeance Proc",
		ActionID: core.ActionID{SpellID: 20059},
		Duration: time.Second * 8,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexHoly] *= vengeanceMultiplier
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= vengeanceMultiplier
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexHoly] /= vengeanceMultiplier
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] /= vengeanceMultiplier
		},
	})

	paladin.RegisterAura(core.Aura{
		Label:    "Vengeance",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidCrit() {
				procAura.Activate(sim)
			}
		},
	})
}

func (paladin *Paladin) applyVindication() {
	if paladin.Talents.Vindication == 0 {
		return
	}
	//vindicationMultiplier := []float64{1, 1.05, 1.10, 1.15}[paladin.Talents.Vengeance]
	vindicationMultiplier := []*stats.StatDependency{
		paladin.NewDynamicMultiplyStat(stats.AttackPower, 1.00),
		paladin.NewDynamicMultiplyStat(stats.AttackPower, 1.05),
		paladin.NewDynamicMultiplyStat(stats.AttackPower, 1.10),
		paladin.NewDynamicMultiplyStat(stats.AttackPower, 1.15),
	}

	vindicationAura := paladin.RegisterAura(core.Aura{
		Label:    "Vindication Proc",
		ActionID: core.ActionID{SpellID: 26021},
		Duration: time.Second * 30,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			paladin.EnableDynamicStatDep(sim, vindicationMultiplier[0])
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			paladin.EnableDynamicStatDep(sim, vindicationMultiplier[paladin.Talents.Vindication])
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			paladin.DisableDynamicStatDep(sim, vindicationMultiplier[paladin.Talents.Vindication])
		},
	})
	// 	vindicationAuras := paladin.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
	// 		return core.VindicationAura(target, paladin.Talents.Vindication)
	// 	})
	paladin.RegisterAura(core.Aura{
		Label:    "Vindication Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// TODO: Replace with actual proc mask / proc chance
			if result.Landed() && spell.ProcMask.Matches(core.ProcMaskMelee) {
				vindicationAura.Activate(sim)
			}
		},
	})
}

func (paladin *Paladin) applyImprovedLayOnHands() {
	/*
	   if paladin.Talents.ImprovedLayOnHands > 0 {

	   		armorMultiplier := []float64{1, 1.15, 1.3}[paladin.Talents.ImprovedLayOnHands]
	   		auraID := []int32{0, 20233, 20236}[paladin.Talents.ImprovedLayOnHands]

	   		paladin.RegisterAura(core.Aura{
	   			Label:    "Lay on Hands",
	   			ActionID: core.ActionID{SpellID: auraID},
	   			Duration: time.Minute * 2,
	   			OnGain: func(aura *core.Aura, sim *core.Simulation) {
	   				paladin.ApplyDynamicEquipScaling(sim, stats.Armor, armorMultiplier)
	   			},
	   			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
	   				paladin.RemoveDynamicEquipScaling(sim, stats.Armor, armorMultiplier)
	   			},
	   			OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
	   				if spell.SpellCode == SpellCode_PaladinLayOnHands {
	   					aura.Activate(sim)
	   				}
	   			},
	   		})
	   	}
	*/
}

// ---------------------------------------------------------------------
// Forever trait-tree talents (talents/paladin.json, build 1.60.1.70009).
//
// Every number below is the client's own rank description
// (data/builds/1.60.1.70009/talents/paladin.json's ranks[].description),
// never a vanilla or SoD figure. A lookup table is used wherever the
// client's own ranks are not a clean multiple of rank 1 (ChampionOfTheLight,
// SanctifiedJudgement's proc chance, PurifyingPower's cooldown reduction);
// everything else is rank * a per-rank constant.
// ---------------------------------------------------------------------

const (
	// Improved Seals (node 105334, Holy tier 1 col 2): "Increases the
	// damage done by your Seals and Judgements by 5%/10%/15%."
	improvedSealsDamagePctPerRank = 0.05

	// Reverence (node 110871, Holy tier 2 col 1): "Allows 10%/20%/30% of
	// your Mana regeneration to continue while casting." Forever's
	// equivalent of vanilla's Arcane Meditation, for a Paladin instead
	// of a Mage; see sim/mage/talents.go's arcaneMeditationRegenWhileCasting.
	reverenceRegenWhileCastingPctPerRank = 0.10

	// Divine Precision (node 105324, Holy tier 4 col 0): "Improves your
	// chance to hit with Holy spells by 6%/12%/18%."
	divinePrecisionHitPctPerRank = 6.0

	// Consecrated Ground (node 110872, Holy tier 4 col 2): "Gives your
	// Holy spells 5%/10% increased damage against the first 4 enemies
	// that enter your Consecration." Approximated as a flat damage bonus
	// on Consecration's OWN damage, rather than a per-enemy-entry debuff
	// tracked across up to 4 targets: every encounter this engine sims
	// is at or below that cap (core.Encounter's target count), so "the
	// first 4 enemies" and "every enemy standing in it" coincide, and
	// tracking "entered" versus "already inside" separately would
	// change nothing a Patchwerk-style fight can observe.
	consecratedGroundDamagePctPerRank = 0.05

	// Holy Conduit (node 105704, Retribution tier 1 col 1): "Reduces the
	// mana cost of your Consecration, Holy Wrath, Exorcism, and Hammer
	// of Wrath spells by 20%/40%." (Also reduces Cleanse and Purify's
	// mana cost; this package registers neither spell, so there is
	// nothing for that half to apply to.)
	holyConduitCostPctPerRank int64 = -20

	// Instrument of Law (node 110880, Retribution tier 5 col 2):
	// "Reduces the cast time of your Hammer of Wrath by 0.5/1 sec, and
	// reduces all threat you generate by 10%/20% while Righteous Fury is
	// not active." Only the cast-time half changes a DPS number; this
	// sim has no threat model for the second half to affect.
	instrumentOfLawCastTimePerRank = -500 * time.Millisecond

	// Sanctified Judgement (node 105701, Retribution tier 2 col 1):
	// "Gives your Judgement ability a 33%/66%/100% chance to return
	// 20%/40%/60% of the Mana cost of the judged seal." "The judged
	// seal" is paladin.currentSealSpell, the seal-cast spell that
	// activated the aura Judgement is about to consume, read BEFORE
	// judgement.go's castSpecificJudgement deactivates it.
	sanctifiedJudgementActionID = 1311074
)

var sanctifiedJudgementChancePerRank = [4]float64{0, 0.33, 0.66, 1.00}
var sanctifiedJudgementRefundPctPerRank = [4]float64{0, 0.20, 0.40, 0.60}

// championOfTheLightSpellPowerPctPerRank: node 110882, Retribution tier 5
// col 1: "Increases your spell damage and healing by up to
// 20%/40%/60% of your Intellect." (the live text and Blizzard's 1 October
// 2026 notes; the earlier client table read 33%/66%/100%).
// Modelled the same way Forever's Arcane Mind reaches Mage spellpower
// (sim/mage/talents.go): a stat dependency from Intellect, here into
// stats.SpellPower, which Spell.GetSchoolDamage (sim/core/spell_result.go)
// adds to every non-physical school including Holy.
var championOfTheLightSpellPowerPctPerRank = [4]float64{0, 0.20, 0.40, 0.60}

// purifyingPowerCooldownPctPerRank: node 105327, Holy tier 2 col 2:
// "Reduces the mana cost of your Cleanse and Purify spells by 10%/20%
// and reduces the cooldown of your Exorcism and Holy Wrath spells by
// 17%/33%." Only the cooldown half is modelled, for the same reason as
// Holy Conduit's Cleanse/Purify half above. The two ranks are not a
// clean multiple of each other (17, 33, not 17, 34), so a lookup table.
var purifyingPowerCooldownPctPerRank = [3]int64{0, -17, -33}

// applyDeclarativeTalents is every Forever trait-tree talent that is a
// pure modifier on a set of spells: as core.SpellModConfig these read
// against the client's own rank text line by line, so a weekly number
// change is a one-line edit. Talents with state, a timer or a proc keep
// their own function (applySanctifiedJudgement, registerTwistOfLight,
// below).
func (paladin *Paladin) applyDeclarativeTalents() {
	t := paladin.Talents

	if t.ImprovedSeals > 0 {
		paladin.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_DamageDone_Pct,
			ClassMask:  PaladinSpellMaskSealsAndJudgementsDamage,
			FloatValue: 1 + improvedSealsDamagePctPerRank*float64(t.ImprovedSeals),
		})
	}

	if t.Reverence > 0 {
		paladin.PseudoStats.SpiritRegenRateCasting += reverenceRegenWhileCastingPctPerRank * float64(t.Reverence)
	}

	if t.PurifyingPower > 0 {
		paladin.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_Cooldown_Multi_Flat,
			ClassMask: PaladinSpellMaskExorcism | PaladinSpellMaskHolyWrath,
			IntValue:  purifyingPowerCooldownPctPerRank[t.PurifyingPower],
		})
	}

	if t.DivinePrecision > 0 {
		paladin.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_BonusHit_Flat,
			School:     core.SpellSchoolHoly,
			FloatValue: divinePrecisionHitPctPerRank * float64(t.DivinePrecision) * core.HitRatingPerHitChance,
		})
	}

	if t.ConsecratedGround > 0 {
		paladin.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_DamageDone_Pct,
			ClassMask:  PaladinSpellMaskConsecration,
			FloatValue: 1 + consecratedGroundDamagePctPerRank*float64(t.ConsecratedGround),
		})
	}

	if t.HolyConduit > 0 {
		paladin.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Pct,
			ClassMask: PaladinSpellMaskHolyConduitCost,
			IntValue:  holyConduitCostPctPerRank * int64(t.HolyConduit),
		})
	}

	if t.InstrumentOfLaw > 0 {
		paladin.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_CastTime_Flat,
			ClassMask: PaladinSpellMaskHammerOfWrath,
			TimeValue: instrumentOfLawCastTimePerRank * time.Duration(t.InstrumentOfLaw),
		})
	}

	if t.ChampionOfTheLight > 0 {
		paladin.AddStatDependency(stats.Intellect, stats.SpellPower, championOfTheLightSpellPowerPctPerRank[t.ChampionOfTheLight])
	}
}

// applySanctifiedJudgement sets up the mana-return metrics Judgement
// (judgement.go) rolls against on every cast; see
// trySanctifiedJudgementManaReturn.
func (paladin *Paladin) applySanctifiedJudgement() {
	if paladin.Talents.SanctifiedJudgement == 0 {
		return
	}
	paladin.sanctifiedJudgementManaMetrics = paladin.NewManaMetrics(core.ActionID{SpellID: sanctifiedJudgementActionID})
}

// trySanctifiedJudgementManaReturn is called from judgement.go's
// ApplyEffects on every Judgement cast, after the specific judgement
// spell lands but before currentSealSpell could change. It is a no-op
// when the talent is untaken, so judgement.go does not need its own rank
// check.
func (paladin *Paladin) trySanctifiedJudgementManaReturn(sim *core.Simulation) {
	rank := paladin.Talents.SanctifiedJudgement
	if rank == 0 || paladin.currentSealSpell == nil || paladin.currentSealSpell.Cost == nil {
		return
	}
	if !sim.Proc(sanctifiedJudgementChancePerRank[rank], "Sanctified Judgement") {
		return
	}
	refund := paladin.currentSealSpell.Cost.BaseCost * sanctifiedJudgementRefundPctPerRank[rank]
	paladin.AddMana(sim, refund, paladin.sanctifiedJudgementManaMetrics)
}

// registerTwistOfLight implements Twist of Light (node 105692,
// Retribution tier 6 col 1): "Reduces the Mana cost of your Seal spells
// by 20%, and when you replace your Seal of Command, Seal of
// Righteousness, Seal of Fury, or Seal of Justice with a different Seal,
// gain an Echo of that Seal. Your next melee attack applies the replaced
// Seal's effects, consuming the Echo."
//
// Seal of Justice is not registered anywhere in this package, so the
// Echo this grants is of Seal of Command, Seal of Righteousness or Seal
// of Fury - whichever this build's seal swap actually crosses. Seal of the
// Crusader is not in the talent's own list (it has no weapon-swing proc
// for an Echo to replay), so swapping into or out of it never grants or
// consumes one; grantEchoOfSeal (paladin.go) returns early for it.
func (paladin *Paladin) registerTwistOfLight() {
	if !paladin.Talents.TwistOfLight {
		return
	}

	paladin.AddStaticMod(core.SpellModConfig{
		Kind:      core.SpellMod_PowerCost_Pct,
		ClassMask: PaladinSpellMaskSealCast,
		IntValue:  -20,
	})

	paladin.echoOfSealAura = paladin.RegisterAura(core.Aura{
		Label:    "Echo of Seal",
		ActionID: core.ActionID{SpellID: 1310735},
		Duration: core.NeverExpires,
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if paladin.pendingEchoOfSealProc == nil || !result.Landed() || !spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) {
				return
			}
			proc := paladin.pendingEchoOfSealProc
			paladin.pendingEchoOfSealProc = nil
			aura.Deactivate(sim)
			proc.Cast(sim, result.Target)
		},
	})
}

// grantEchoOfSeal is called from applySeal (paladin.go), before
// currentSeal is overwritten, with the Aura being replaced.
func (paladin *Paladin) grantEchoOfSeal(sim *core.Simulation, oldSeal *core.Aura) {
	var proc *core.Spell
	switch {
	case slices.Contains(paladin.aurasSoR, oldSeal):
		proc = paladin.sealOfRighteousnessProc
	case slices.Contains(paladin.aurasSoC, oldSeal):
		proc = paladin.sealOfCommandProc
	case slices.Contains(paladin.aurasSoF, oldSeal):
		proc = paladin.sealOfFuryProc
	default:
		// Seal of the Crusader, or any seal rank not in either slice:
		// no Echo. See registerTwistOfLight's doc comment.
		return
	}
	if proc == nil {
		return
	}

	paladin.pendingEchoOfSealProc = proc
	paladin.echoOfSealAura.Activate(sim)
}

// markUnmodeledForeverTalents names every Forever trait-tree talent this
// package does not model, with the one-line reason, so a future reader
// can tell "not implemented" from "not noticed". Grouped by why, not by
// tree position.
func (paladin *Paladin) markUnmodeledForeverTalents() {
	t := paladin.Talents

	// Healing talents: this package has no healer rotation and sims a
	// Patchwerk-style fight where nothing the Paladin heals matters to
	// DPS, so none of these change a simmed number.
	_, _, _, _ = t.HealingLight, t.SpiritualFocus, t.InfusionOfLight, t.Illumination

	// Crowd control and escape effects a single-boss tank sim never
	// meets: Guardian's Favor shortens Blessing of Protection's cooldown
	// and lengthens Blessing of Freedom (neither is registered), and
	// Unyielding Faith shortens Fear and Disorient. The rest of the
	// Protection tree is modelled in tank_talents.go and its abilities.
	_, _ = t.GuardiansFavor, t.UnyieldingFaith

	// Crowd control and its own cooldown: Repentance incapacitates
	// (never cast on a Patchwerk-style boss), and Improved Hammer of
	// Justice only shortens a stun's cooldown, not a damage ability's.
	_, _ = t.Repentance, t.ImprovedHammerOfJustice

	// Utility with no DPS reading: a silence/interrupt immunity window,
	// a movement speed bonus, and a healing capstone whose enemy-damage
	// option (182 Holy damage) is a minor side effect of a group-support
	// cooldown this package's Retribution rotation does not cast.
	_, _, _ = t.VoiceOfTruth, t.PursuitOfJustice, t.LightsVigil

	// Eye for an Eye reflects a fraction of a melee crit taken back at
	// the attacker - a tank/defensive proc gated on being hit, not on
	// anything the Paladin casts.
	_ = t.EyeForAnEye
}
