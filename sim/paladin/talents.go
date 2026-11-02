package paladin

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

func (paladin *Paladin) ApplyTalents() {
	// TODO: paladin.AddStat(stats.RangedHit, float64(paladin.Talents.Precision)*core.HitRatingPerHitChance)

	paladin.applyConviction()

	// These are no-op if untalented.
	paladin.MultiplyStat(stats.Strength, 1.0+0.02*float64(paladin.Talents.DivineStrength))
	paladin.MultiplyStat(stats.Intellect, 1.0+0.02*float64(paladin.Talents.DivineIntellect))

	paladin.AddStat(stats.Parry, 1*float64(paladin.Talents.Deflection))

	paladin.applyWeaponSpecialization()
	if paladin.Talents.Vengeance > 0 {
		paladin.applyVengeance()
	}

	paladin.applyProtectionTalents()
	paladin.applyImprovedLayOnHands()

	// Client trait-tree talents (talents/paladin.json, build
	// 1.60.1.70009). Pure spell modifiers are declarative config, below;
	// anything with state, a timer or a proc keeps its own function.
	paladin.applyDeclarativeTalents()
	paladin.applyHealingTalents()
	paladin.applySanctifiedJudgement()
	paladin.markUnmodeledForeverTalents()
}

func (paladin *Paladin) improvedSoR() float64 {
	// FOREVER: Improved Seal of Righteousness is not in the client's trees.
	// return []float64{1, 1.03, 1.06, 1.09, 1.12, 1.15}[paladin.Talents.ImprovedSealOfRighteousness]
	return 1
}

// benedictionMaxRanks and benedictionCostReductionPerRank: Benediction is
// "Reduces the Mana cost of all instant cast spells and abilities by 2%" a
// rank, 10% at 5. The client's one spell row (20101, a -10 cost modifier at
// the last rank) states the total.
const (
	benedictionMaxRanks                   = 5
	benedictionCostReductionPerRank       = 2
	benedictionClientSpellID              = 20101
	benedictionFullCostPercent      int32 = 100
)

// benediction is the percent of its base cost an instant spell costs.
func (paladin *Paladin) benediction() int32 {
	return benedictionFullCostPercent - benedictionCostReductionPerRank*paladin.Talents.Benediction
}

// Conviction (node 105703, 3 ranks): "Improves your chance to get a
// critical strike with all spells and attacks by 2%" a rank, 6% at rank 3
// (client 1.60.1.70707; it was 1% melee crit a rank over 5 ranks). Its
// aura 290 carries a SpellEquippedItems row (melee weapons, subclass mask
// 173555), as Cruelty's does, and the beta logs show it on the weapon only:
// Retribution paladins crit Consecration 6.5% of 84,212 ticks against 8.1%
// for the rest, while their melee crits 10.1% against 6.2% (foreverlogs.gg,
// 105 paladins, levels 19 to 35). So it is melee and ranged crit.
const convictionCritPctPerRank = 2.0

func (paladin *Paladin) applyConviction() {
	paladin.AddStat(stats.MeleeCrit, convictionCritPctPerRank*float64(paladin.Talents.Conviction)*core.CritRatingPerCritChance)
}

// Two-Handed Weapon Specialization (node 105697, 3 ranks): "Increases the
// physical damage you deal by 2%, and reduces all threat you generate by
// 7% while a two-handed melee weapon is equipped", 4%/14% and 6%/20% at
// ranks 2 and 3 (client 1.60.1.70707). The text names physical damage, so
// it no longer reaches the Holy weapon attacks (seal procs, Judgement of
// Command, Holy Strike, Hammer of the Righteous) that "damage you deal with
// two-handed melee weapons" did before 70707. The two-hander condition is
// read as covering both halves, as the talent's name and its 70658 text
// say (unconfirmed for the damage half).
const twoHandedWeaponSpecializationDamagePerRank = 0.02

var twoHandedWeaponSpecializationThreatReduction = [4]float64{0, 0.07, 0.14, 0.20}

// applyWeaponSpecialization scales physical damage and threat while a
// two-hander is in the main hand.
func (paladin *Paladin) applyWeaponSpecialization() {
	rank := paladin.Talents.TwoHandedWeaponSpecialization
	if rank == 0 || !paladin.has2hEquipped() {
		return
	}
	paladin.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= 1 + twoHandedWeaponSpecializationDamagePerRank*float64(rank)
	paladin.PseudoStats.ThreatMultiplier *= 1 - twoHandedWeaponSpecializationThreatReduction[rank]
}

// Forever's Vengeance (1.60.1.70291 talents/paladin.json): "Increases your
// Physical and Holy damage dealt by 1/2/3% for 30 sec after landing a
// non-periodic critical strike. Stacks up to 3 times"; an aura's amount
// applies once a stack, so 3/3 with three stacks is +9%. Vanilla's 3% a
// rank for 8 sec with no stacks is not Forever's.
const (
	vengeanceDamagePerRank = 0.01
	vengeanceDuration      = 30 * time.Second
	vengeanceMaxStacks     = 3
)

// vengeanceMultiplier is the Physical and Holy damage multiplier Vengeance
// grants at a talent rank and stack count.
func vengeanceMultiplier(rank int32, stacks int32) float64 {
	return 1 + vengeanceDamagePerRank*float64(rank)*float64(stacks)
}

// vengeanceBuffSpellID is Vengeance's buff, 20050 (triggered by 20049) at
// every rank in 1.60.1.70707.
const vengeanceBuffSpellID = 20050

func (paladin *Paladin) applyVengeance() {
	if paladin.Talents.Vengeance == 0 {
		return
	}

	rank := paladin.Talents.Vengeance

	procAura := paladin.RegisterAura(core.Aura{
		Label:     "Vengeance Proc",
		ActionID:  core.ActionID{SpellID: vengeanceBuffSpellID},
		Duration:  vengeanceDuration,
		MaxStacks: vengeanceMaxStacks,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
			ratio := vengeanceMultiplier(rank, newStacks) / vengeanceMultiplier(rank, oldStacks)
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexHoly] *= ratio
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= ratio
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
				procAura.AddStack(sim)
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
	// Improved Seals (node 105334, Holy tier 1 col 2, 5 ranks): "Increases
	// the damage done by your Seals and Judgements by 3%", 15% at rank 5
	// (client 1.60.1.70707; 5% a rank over 3 ranks before).
	improvedSealsDamagePctPerRank = 0.03

	// Reverence (node 110871, Holy tier 1 col 1, 2 ranks): "Allows 15%/30%
	// of your Mana regeneration to continue while casting" (client
	// 1.60.1.70707; 10% a rank over 3 ranks before). Forever's
	// equivalent of vanilla's Arcane Meditation, for a Paladin instead
	// of a Mage; see sim/mage/talents.go's arcaneMeditationRegenWhileCasting.
	reverenceRegenWhileCastingPctPerRank = 0.15

	// Divine Precision (node 105324, Holy tier 3 col 0, 2 ranks): "Improves
	// your chance to hit with Holy spells by 4%/8%, and reduces the
	// duration of Silence effects on you by 10%/20%" (client 1.60.1.70707;
	// 6% a rank over 3 ranks before). The silence half meets nothing here.
	divinePrecisionHitPctPerRank = 4.0

	// Consecrated Ground (node 110872, Holy tier 4 col 0): "Gives your
	// Holy spells 5%/10% increased damage against the first 4 enemies
	// that enter your Consecration." Approximated as a flat damage bonus
	// on Consecration's OWN damage, rather than a per-enemy-entry debuff
	// tracked across up to 4 targets: every encounter this engine sims
	// is at or below that cap (core.Encounter's target count), so "the
	// first 4 enemies" and "every enemy standing in it" coincide, and
	// tracking "entered" versus "already inside" separately would
	// change nothing a Patchwerk-style fight can observe.
	consecratedGroundDamagePctPerRank = 0.05

	// Holy Conduit (node 105704, Holy tier 2 col 2 since 1.60.1.70707; it
	// was a Retribution talent before): "Reduces the
	// mana cost of your Consecration, Holy Wrath, Exorcism, and Hammer
	// of Wrath spells by 20%/40%." (Also reduces Cleanse and Purify's
	// mana cost; this package registers neither spell, so there is
	// nothing for that half to apply to.)
	holyConduitCostPctPerRank int64 = -20

	// Sanctified Judgement (node 105701, Retribution tier 1 col 2):
	// "Gives your Judgement ability a 33%/66%/100% chance to return
	// 20%/40%/60% of the Mana cost of the judged seal." "The judged
	// seal" is paladin.currentSealSpell, the seal-cast spell that
	// activated the aura Judgement is about to consume, read BEFORE
	// judgement.go's castSpecificJudgement deactivates it.
	sanctifiedJudgementActionID = 1311074
)

var sanctifiedJudgementChancePerRank = [4]float64{0, 0.33, 0.66, 1.00}
var sanctifiedJudgementRefundPctPerRank = [4]float64{0, 0.20, 0.40, 0.60}

// championOfTheLightPctPerRank and championOfTheLightCastTimePerRank: node
// 110882, Retribution tier 5 col 2, 5 ranks since 1.60.1.70707: "Increases
// your spell damage by up to 12% of your Intellect, and reduces the cast
// time of your Hammer of Wrath by 0.2 sec" a rank, 60% and 1.0 sec at rank
// 5 (the removed Instrument of Law's cast-time half moved here; 70532 had
// 20/40/60% over 3 ranks).
var championOfTheLightPctPerRank = [6]float64{0, 0.12, 0.24, 0.36, 0.48, 0.60}

const championOfTheLightCastTimePerRank = 200 * time.Millisecond

// applyChampionOfTheLight grants the share to spell power (damage) only, for
// every spec: the 1.60.1.70707 row 1311084 has one stat aura, 174 (spell
// damage of an Intellect percent), and no 175 (healing); the 31 October
// hotfix push adds only the Hammer of Wrath cast-time effect (1367233,
// modelled in applyDeclarativeTalents); and Wowhead Forever's live text
// names spell damage alone.
func (paladin *Paladin) applyChampionOfTheLight() {
	share := championOfTheLightPctPerRank[paladin.Talents.ChampionOfTheLight]
	paladin.AddStatDependency(stats.Intellect, stats.SpellPower, share)
}

// purifyingPowerCooldownPctPerRank: node 105327, Holy tier 1 col 3:
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
// their own function (applySanctifiedJudgement below, Unsealed Wrath in
// unsealed_wrath.go).
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

	if t.ChampionOfTheLight > 0 {
		paladin.applyChampionOfTheLight()
		paladin.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_CastTime_Flat,
			ClassMask: PaladinSpellMaskHammerOfWrath,
			TimeValue: -championOfTheLightCastTimePerRank * time.Duration(t.ChampionOfTheLight),
		})
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

// markUnmodeledForeverTalents names every Forever trait-tree talent this
// package does not model, with the one-line reason, so a future reader
// can tell "not implemented" from "not noticed". Grouped by why, not by
// tree position.
func (paladin *Paladin) markUnmodeledForeverTalents() {
	t := paladin.Talents

	// Crowd control and escape effects a single-boss sim never meets:
	// Guardian's Favor shortens Blessing of Protection's cooldown and
	// lengthens Blessing of Freedom (neither is registered); Unyielding
	// Faith (Fear and Disorient) and Stoicism (Stun and Charm) shorten
	// effects on the paladin; Turn Evil speeds up Turn Undead.
	_, _, _, _ = t.GuardiansFavor, t.UnyieldingFaith, t.Stoicism, t.TurnEvil

	// Crowd control and its own cooldown: Repentance incapacitates
	// (never cast on a Patchwerk-style boss), and Improved Hammer of
	// Justice only shortens a stun's cooldown, not a damage ability's.
	_, _ = t.Repentance, t.ImprovedHammerOfJustice

	// Utility with no healing or DPS reading: a movement speed bonus.
	_ = t.PursuitOfJustice

	// Eye for an Eye reflects a fraction of a melee crit taken back at
	// the attacker - a tank/defensive proc gated on being hit, not on
	// anything the Paladin casts.
	_ = t.EyeForAnEye

	// Vindication (1.60.1.70707) only lowers the target's attack power;
	// its client text prints an unresolved formula, "(3 /- 3 * 6)", and
	// the self attack power half it had before is gone.
	_ = t.Vindication

}
