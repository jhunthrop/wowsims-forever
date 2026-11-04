package rogue

import (
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

func (rogue *Rogue) ApplyTalents() {
	rogue.applyRuthlessness()
	rogue.applyRelentlessStrikes()
	rogue.applySealFate()
	rogue.applyWeaponSpecializations()
	rogue.applyWeaponExpertise()
	rogue.applyInitiative()
	rogue.applyPuncturingWounds()
	rogue.applyHackAndSlash()
	rogue.applyFlawlessExecution()
	rogue.applyEfficientStrikes()
	rogue.applySetup()

	// FOREVER: talents with no DPS lever on a Patchwerk fight. Each is
	// commented with why, so the reader can tell "not implemented" from
	// "not noticed" (same convention as mage/talents.go).

	// Improved Gouge lengthens a CC ability's duration; Gouge has no
	// spell file anywhere in this package, so there is nothing for it
	// to lengthen.
	_ = rogue.Talents.ImprovedGouge

	// Remorseless Attacks procs 20%/40% crit on the next builder after
	// killing a non-trivial enemy. No hook in sim/core fires on a unit's
	// death (an alias-aware grep for OnKill/OnUnitDeath/OnTargetDeath
	// across sim/core turns up nothing), and a Patchwerk fight's only
	// kill is the boss at the very end of the encounter, after which
	// nothing else casts - the same "needs a killing blow a Patchwerk
	// fight never produces" gap mage/talents.go documents for Wake of
	// Fire's kill-triggered half. A core.Unit "died" callback is the
	// missing piece, and that is a sim/core change, out of scope for
	// this package alone.
	_ = rogue.Talents.RemorselessAttacks

	// Improved Kidney Shot buffs damage against a target stunned by
	// Kidney Shot, but Kidney Shot itself has no spell file in this
	// package (no kidney_shot.go, nothing registers or casts it) -
	// nothing here ever applies the stun this debuff keys off.
	_ = rogue.Talents.ImprovedKidneyShot

	// Endurance halves the cooldown of Sprint and Evasion. Evasion
	// (evasion.go) is a defensive dodge cooldown with no damage lever,
	// and Sprint has no spell file in this package at all. Neither
	// changes a DPS number.
	_ = rogue.Talents.Endurance

	// Improved Sprint gives Sprint a chance to clear movement-impairing
	// effects; Sprint itself has no spell file here.
	_ = rogue.Talents.ImprovedSprint

	// Improved Kick adds a silence chance to an interrupt. Kick has no
	// spell file in this package, and a Patchwerk boss casts nothing to
	// interrupt in the first place.
	_ = rogue.Talents.ImprovedKick

	// Camouflage reduces Stealth's move-speed penalty and cooldown.
	// stealth.go's own comment already says the move-speed penalty is
	// not modeled, and the Stealth this package registers is a free,
	// instant, pre-pull-only cast with no registered cooldown for a
	// rank to shorten.
	_ = rogue.Talents.Camouflage

	// Master of Deception and Heightened Senses are Stealth-detection or
	// being-hit-by-ranged/spells mitigation: nothing in this sim models an
	// enemy detecting a stealthed rogue or casting at one, so neither has
	// a number to change.
	_, _ = rogue.Talents.MasterOfDeception, rogue.Talents.HeightenedSenses

	// Quick Getaway (1310728) raises the Sap target cap and triggers a
	// 1329282 run-speed buff after a Sap. Sap has no spell file in this
	// package and a Patchwerk fight saps nothing.
	_ = rogue.Talents.QuickGetaway

	// Dirty Tricks discounts Sap and Blind. Neither has a spell file in
	// this package: both are pre-pull/CC utility, not a Patchwerk
	// rotation ability.
	_ = rogue.Talents.DirtyTricks

	rogue.AddStat(stats.Dodge, 1*float64(rogue.Talents.LightningReflexes))
	rogue.AddStat(stats.Parry, deflectionParryPerRank*float64(rogue.Talents.Deflection))
	rogue.AddStat(stats.Crit, 1*float64(rogue.Talents.Malice))
	rogue.AddStat(stats.Hit, 1*float64(rogue.Talents.Precision))
	// Serrated Blades' armor half is aura 280 (SpellEffect 690915): a share
	// of the target's armor, not flat armor penetration.
	rogue.PseudoStats.ArmorIgnorePercent += 0.01 * serratedBladesArmorPenPctPerRank * float64(rogue.Talents.SerratedBlades)
	rogue.AutoAttacks.OHConfig().DamageMultiplier *= rogue.dwsMultiplier()

	/*
		if rogue.Talents.Deadliness > 0 {
			rogue.MultiplyStat(stats.AttackPower, 1.0+0.02*float64(rogue.Talents.Deadliness))
		}
	*/

	rogue.registerColdBloodCD()
	rogue.registerBladeFlurryCD()
	rogue.registerAdrenalineRushCD()
	rogue.registerPreparationCD()
	rogue.registerPremeditation()
	rogue.registerGhostlyStrikeSpell()
	rogue.applyRiposte()
}

// dwsMultiplier returns the offhand damage multiplier
func (rogue *Rogue) dwsMultiplier() float64 {
	return 1 + dualWieldSpecializationPerRank*float64(rogue.Talents.DualWieldSpecialization)
}

func (rogue *Rogue) applyRuthlessness() {
	if rogue.Talents.Ruthlessness == 0 {
		return
	}

	procChance := 0.2 * float64(rogue.Talents.Ruthlessness)
	cpMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 14161})
	rogue.OnComboPointsSpent(func(sim *core.Simulation, spell *core.Spell, comboPoints int32) {
		if sim.Proc(procChance, "Ruthlessness") {
			rogue.AddComboPointsIgnoreTarget(sim, 1, cpMetrics)
		}
	})
}

// applyRelentlessStrikes: 14179 e0 triggers 14181 (25 Energy); the chance
// per combo point is the trait curve (TraitDefinitionEffectPoints 25222,
// curve 128113: 4/8/12/16/20%).
func (rogue *Rogue) applyRelentlessStrikes() {
	rank := rogue.Talents.RelentlessStrikes
	if rank == 0 {
		return
	}

	chancePerComboPoint := relentlessStrikesChancePerComboPoint(rank)
	cpMetrics := rogue.NewEnergyMetrics(core.ActionID{SpellID: 14179})
	rogue.OnComboPointsSpent(func(sim *core.Simulation, spell *core.Spell, comboPoints int32) {
		if sim.Proc(chancePerComboPoint*float64(comboPoints), "RelentlessStrikes") {
			rogue.AddEnergy(sim, relentlessStrikesEnergy, cpMetrics)
		}
	})
}

// relentlessStrikesChancePerComboPoint is the restore chance per combo
// point at a Relentless Strikes rank.
func relentlessStrikesChancePerComboPoint(rank int32) float64 {
	return relentlessStrikesChancePerRank * float64(max(0, min(rank, relentlessStrikesMaxRank)))
}

// Cold Blood talent
func (rogue *Rogue) registerColdBloodCD() {
	if !rogue.Talents.ColdBlood {
		return
	}

	actionID := core.ActionID{SpellID: 14177}

	coldBloodAura := rogue.RegisterAura(core.Aura{
		Label:    "Cold Blood",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range rogue.Spellbook {
				if spell.Flags.Matches(SpellFlagColdBlooded) {
					spell.BonusCritRating += 100 * core.CritRatingPerCritChance
				}
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range rogue.Spellbook {
				if spell.Flags.Matches(SpellFlagColdBlooded) {
					spell.BonusCritRating -= 100 * core.CritRatingPerCritChance
				}
			}
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// Mutilate's main-hand hit leaves Cold Blood up for the
			// off-hand hit that follows it in the same cast.
			if spell.Flags.Matches(SpellFlagColdBlooded) && spell != rogue.MutilateMH {
				aura.Deactivate(sim)
			}
		},
	})

	rogue.ColdBlood = rogue.RegisterSpell(core.SpellConfig{
		ActionID: actionID,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Minute * 3,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			coldBloodAura.Activate(sim)
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell: rogue.ColdBlood,
		Type:  core.CooldownTypeDPS,
	})
}

// sealFateChance is Seal Fate's rank -> proc chance, index 0 unused: the
// trait curve (TraitDefinitionEffectPoints 23365, curve 83040: 33/66/100).
var sealFateChance = [4]float64{0, 0.33, 0.66, 1.0}

// Seal Fate talent
func (rogue *Rogue) applySealFate() {
	if rogue.Talents.SealFate == 0 {
		return
	}

	procChance := sealFateChance[rankIndex(rogue.Talents.SealFate, sealFateChance[:])]
	cpMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 14195})

	icd := core.Cooldown{
		Timer:    rogue.NewTimer(),
		Duration: 500 * time.Millisecond,
	}

	rogue.RegisterAura(core.Aura{
		Label:    "Seal Fate",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.Flags.Matches(SpellFlagBuilder) {
				return
			}

			if !result.Outcome.Matches(core.OutcomeCrit) {
				return
			}

			if icd.IsReady(sim) && sim.Proc(procChance, "Seal Fate") {
				rogue.AddComboPoints(sim, 1, result.Target, cpMetrics)
				icd.Use(sim)
			}
		},
	})
}

// initiativeComboPointChance is Initiative's rank -> proc chance, index 0
// unused: the client's 33%/67%/100% rank text.
var initiativeComboPointChance = [4]float64{0, 0.33, 0.67, 1.0}

// Initiative talent
func (rogue *Rogue) applyInitiative() {
	if rogue.Talents.Initiative == 0 {
		return
	}

	procChance := initiativeComboPointChance[rankIndex(rogue.Talents.Initiative, initiativeComboPointChance[:])]
	cpMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 13980})

	rogue.RegisterAura(core.Aura{
		Label:    "Initiative",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell == rogue.Garrote || spell == rogue.Ambush || spell == rogue.CheapShot {
				if result.Landed() {
					if sim.Proc(procChance, "Initiative") {
						rogue.AddComboPoints(sim, 1, result.Target, cpMetrics)
					}
				}
			}
		},
	})
}

// Rogue weapon specialization talents. Bonus is shown if the main hand is specialized, but not if off hand only
func (rogue *Rogue) applyWeaponSpecializations() {
	/*
		// Sword specialization. Implemented in 'sword_specialization.go'
		if swordSpec := rogue.Talents.SwordSpecialization; swordSpec > 0 {
			if mask := rogue.GetProcMaskForTypes(proto.WeaponType_WeaponTypeSword); mask != core.ProcMaskUnknown {
				rogue.registerSwordSpecialization(mask)
			}
		}

		// Dagger Specialization
		if daggerSpec := rogue.Talents.DaggerSpecialization; daggerSpec > 0 {
			switch rogue.GetProcMaskForTypes(proto.WeaponType_WeaponTypeDagger) {
			case core.ProcMaskMelee:
				rogue.AddStat(stats.Crit, core.CritRatingPerCritChance*float64(daggerSpec))
			case core.ProcMaskMeleeMH:
				// the default character pane displays critical strike chance for main hand only
				rogue.AddStat(stats.Crit, core.CritRatingPerCritChance*float64(daggerSpec))
				rogue.OnSpellRegistered(func(spell *core.Spell) {
					if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
						spell.BonusCritRating -= core.CritRatingPerCritChance * float64(daggerSpec)
					}
				})
			case core.ProcMaskMeleeOH:
				rogue.OnSpellRegistered(func(spell *core.Spell) {
					if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
						spell.BonusCritRating += core.CritRatingPerCritChance * float64(daggerSpec)
					}
				})
			}
		}

		// Fist Weapon Specialization. Same as above but for fists
		if fistSpec := rogue.Talents.FistWeaponSpecialization; fistSpec > 0 {
			switch rogue.GetProcMaskForTypes(proto.WeaponType_WeaponTypeFist) {
			case core.ProcMaskMelee:
				rogue.AddStat(stats.Crit, core.CritRatingPerCritChance*float64(fistSpec))
			case core.ProcMaskMeleeMH:
				// the default character pane displays critical strike chance for main hand only
				rogue.AddStat(stats.Crit, core.CritRatingPerCritChance*float64(fistSpec))
				rogue.OnSpellRegistered(func(spell *core.Spell) {
					if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
						spell.BonusCritRating -= core.CritRatingPerCritChance * float64(fistSpec)
					}
				})
			case core.ProcMaskMeleeOH:
				rogue.OnSpellRegistered(func(spell *core.Spell) {
					if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
						spell.BonusCritRating += core.CritRatingPerCritChance * float64(fistSpec)
					}
				})
			}
		}

		// Mace Specialization. Offers weapon skill for Maces and RNG stun (not implemented for being useless on boss)
		if maceSpec := rogue.Talents.MaceSpecialization; maceSpec > 0 {
			if mask := rogue.GetProcMaskForTypes(proto.WeaponType_WeaponTypeMace); mask != core.ProcMaskUnknown {
				rogue.PseudoStats.MacesSkill += float64(maceSpec)
			}
		}
	*/
}

// weaponExpertisePercent is Weapon Expertise's dodge-and-parry reduction
// per the live tree (data/builds/1.60.1.70009/talents/rogue.json): "Reduces
// the chance for your attacks to be Dodged or Parried by 1%" and "2%" for
// its two ranks. Vanilla's +3/+5 weapon skill is gone with the talent text.
func weaponExpertisePercent(points int32) float64 {
	return float64(max(0, min(points, 2)))
}

func (rogue *Rogue) applyWeaponExpertise() {
	if percent := weaponExpertisePercent(rogue.Talents.WeaponExpertise); percent > 0 {
		rogue.AddStat(stats.Expertise, percent*core.ExpertiseRatingPerExpertiseChance)
	}
}

func (rogue *Rogue) registerBladeFlurryCD() {
	if !rogue.Talents.BladeFlurry {
		return
	}

	// TODO verify that this double dips from damage modifiers

	var curDmg float64
	bfHit := rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 22482},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskEmpty, // No proc mask, so it won't proc itself.
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, curDmg, spell.OutcomeAlwaysHit)
		},
	})

	rogue.BladeFlurryAura = rogue.RegisterAura(core.Aura{
		Label:    "Blade Flurry",
		ActionID: core.ActionID{SpellID: 13877},
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyMeleeSpeed(sim, 1.2)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyMeleeSpeed(sim, 1/1.2)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if sim.GetNumTargets() < 2 {
				return
			}

			if result.Damage == 0 || !spell.ProcMask.Matches(core.ProcMaskMelee) {
				return
			}

			// Undo armor reduction to get the raw damage value.
			curDmg = result.Damage / result.ResistanceMultiplier

			bfHit.Cast(sim, rogue.Env.NextTargetUnit(result.Target))
			bfHit.SpellMetrics[result.Target.UnitIndex].Casts--
		},
	})

	cooldownDur := time.Minute * 2
	rogue.BladeFlurry = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_RogueBladeFlurry,
		ClassSpellMask: RogueSpellMaskBladeFlurry,
		ActionID:       core.ActionID{SpellID: 13877},
		Flags:          core.SpellFlagAPL,
		RequiredLevel:  1,

		EnergyCost: core.EnergyCostOptions{
			Cost: 25,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: cooldownDur,
			},
		},

		// See Adrenaline Rush's RelatedSelfBuff comment above.
		RelatedSelfBuff: rogue.BladeFlurryAura,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			rogue.BladeFlurryAura.Activate(sim)
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell:    rogue.BladeFlurry,
		Type:     core.CooldownTypeDPS,
		Priority: core.CooldownPriorityDefault,
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			return bladeFlurryShouldActivate(sim.GetRemainingDuration(), cooldownDur, rogue.BladeFlurryAura.Duration, rogue.SliceAndDiceAura.RemainingDuration(sim))
		},
	})
}

// bladeFlurryShouldActivate fires Blade Flurry at once while another cast
// still fits in the fight; the last one waits for Slice and Dice, but no
// later than the point where the fight has only Blade Flurry's own
// duration left, so a rotation that lets Slice and Dice drop still gets it.
func bladeFlurryShouldActivate(remaining, cooldown, duration, sliceAndDiceRemaining time.Duration) bool {
	if remaining > cooldown+duration {
		return true
	}
	return sliceAndDiceRemaining >= time.Second || remaining <= duration
}

var AdrenalineRushActionID = core.ActionID{SpellID: 13750}

func (rogue *Rogue) registerAdrenalineRushCD() {
	if !rogue.Talents.AdrenalineRush {
		return
	}

	rogue.AdrenalineRushAura = rogue.RegisterAura(core.Aura{
		Label:    "Adrenaline Rush",
		ActionID: AdrenalineRushActionID,
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.AddEnergyRegenMultiplier(1.0)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.AddEnergyRegenMultiplier(-1.0)
		},
	})

	rogue.AdrenalineRush = rogue.RegisterSpell(core.SpellConfig{
		SpellCode: SpellCode_RogueAdrenalineRush,
		ActionID:  AdrenalineRushActionID,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Minute * 5,
			},
		},

		// The conformance report (sim/conformance/compare.go's
		// engineDuration) only reads a spell's duration off
		// RelatedSelfBuff or a current-target Dot; without this, Adrenaline
		// Rush's real 15s buff was invisible to it.
		RelatedSelfBuff: rogue.AdrenalineRushAura,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			rogue.AdrenalineRushAura.Activate(sim)
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell:    rogue.AdrenalineRush,
		Type:     core.CooldownTypeDPS,
		Priority: core.CooldownPriorityBloodlust,
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			return rogue.CurrentEnergy() <= 45.0
		},
	})
}

func (rogue *Rogue) lethality() float64 {
	return lethalityCritBonusPerRank * float64(rogue.Talents.Lethality)
}

// Per-rank talent values below, all read off the client's own rank
// descriptions for build 1.60.1.70009
// (data/builds/1.60.1.70009/talents/rogue.json). A Forever patch that
// changes a number changes a line here and nothing else.
const (
	// Puncturing Wounds: "Increases the critical strike chance of your
	// Backstab by 10%/20%/30% and your Mutilate by 5%/10%/15%, and gives
	// Backstab a 15%/30%/45% chance to add an additional Combo Point."
	puncturingWoundsBackstabCritPerRank = 10.0
	puncturingWoundsMutilateCritPerRank = 5.0
	puncturingWoundsExtraComboPerRank   = 0.15
	// Hack and Slash, per rank: Axe/Sword "1%/2%/3%/4%/5% chance to
	// trigger an extra attack"; Dagger/Fist "increases your critical
	// strike chance by 1%/2%/3%/4%/5%"; Mace "attacks ignore
	// 3%/6%/9%/12%/15% of your target's armor."
	hackAndSlashExtraAttackChancePerRank = 0.01
	hackAndSlashCritPerRank              = 1.0
	hackAndSlashArmorPenPctPerRank       = 3.0
	// Flawless Execution (1 rank): 1310711 e0 aura 107 (cost) on
	// Eviscerate, the trait curve's -15 (TraitDefinitionEffectPoints 25225,
	// curve 128126).
	flawlessExecutionEnergyCostReduction = 15
	// Relentless Strikes: 25 Energy (14181) at 4% per combo point per rank.
	relentlessStrikesChancePerRank = 0.04
	relentlessStrikesMaxRank       = 5
	relentlessStrikesEnergy        = 25
)

// setupComboPointChance is Setup's rank -> proc chance, index 0 unused:
// the trait curve (TraitDefinitionEffectPoints 23403, curve 83009:
// 15/30/45%).
var setupComboPointChance = [4]float64{0, 0.15, 0.30, 0.45}

// isOffHandStrike reports whether spell swings the off-hand weapon: every
// off-hand proc-mask spell, and Shank, which strikes with the off hand but
// carries an empty proc mask because the client suppresses its procs.
func isOffHandStrike(spell *core.Spell) bool {
	return spell.ProcMask.Matches(core.ProcMaskMeleeOH) || spell.Matches(RogueSpellMaskShank)
}

// applyPuncturingWounds gives Backstab and Mutilate's crit bonus and
// Backstab's extra-combo-point chance. The crit bonus is applied once,
// deferred to Env.RegisterPreFinalizeEffect: ApplyTalents runs before
// Initialize (core/character.go's applyAllEffects calls agent.ApplyTalents
// first), so rogue.Backstab/MutilateMH/MutilateOH are all still nil here;
// by the pre-finalize phase every player has finished Initialize.
func (rogue *Rogue) applyPuncturingWounds() {
	rank := rogue.Talents.PuncturingWounds
	if rank == 0 {
		return
	}

	backstabCritBonus := puncturingWoundsBackstabCritPerRank * float64(rank) * core.CritRatingPerCritChance
	mutilateCritBonus := puncturingWoundsMutilateCritPerRank * float64(rank) * core.CritRatingPerCritChance
	extraComboChance := puncturingWoundsExtraComboPerRank * float64(rank)

	rogue.Env.RegisterPreFinalizeEffect(func() {
		if rogue.Backstab != nil {
			rogue.Backstab.BonusCritRating += backstabCritBonus
		}
		if rogue.MutilateMH != nil {
			rogue.MutilateMH.BonusCritRating += mutilateCritBonus
		}
		if rogue.MutilateOH != nil {
			rogue.MutilateOH.BonusCritRating += mutilateCritBonus
		}
	})

	cpMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 1224716})
	rogue.RegisterAura(core.Aura{
		Label:    "Puncturing Wounds",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// rogue.Backstab is read lazily here (every call), unlike the
			// eager BonusCritRating writes above, so it is always the
			// live, Initialize-populated pointer.
			if spell != rogue.Backstab || !result.Landed() {
				return
			}
			if sim.Proc(extraComboChance, "Puncturing Wounds") {
				rogue.AddComboPoints(sim, 1, result.Target, cpMetrics)
			}
		},
	})
}

// applyHackAndSlash gives melee weapon attacks a per-weapon-type benefit.
// Equipment is already applied by the time ApplyTalents runs
// (character.applyEquipment precedes agent.ApplyTalents in
// core/character.go's applyAllEffects), so reading MainHand()/OffHand()
// here is safe, unlike the spell-pointer reads above.
func (rogue *Rogue) applyHackAndSlash() {
	rank := rogue.Talents.HackAndSlash
	if rank == 0 {
		return
	}

	extraAttackChance := hackAndSlashExtraAttackChancePerRank * float64(rank)
	critBonus := hackAndSlashCritPerRank * float64(rank) * core.CritRatingPerCritChance
	armorIgnore := 0.01 * hackAndSlashArmorPenPctPerRank * float64(rank)
	actionID := core.ActionID{SpellID: 13960}

	isAxeOrSword := func(weapon *core.Item) bool {
		return weapon != nil && (weapon.WeaponType == proto.WeaponType_WeaponTypeAxe || weapon.WeaponType == proto.WeaponType_WeaponTypeSword)
	}
	isDaggerOrFist := func(weapon *core.Item) bool {
		return weapon != nil && (weapon.WeaponType == proto.WeaponType_WeaponTypeDagger || weapon.WeaponType == proto.WeaponType_WeaponTypeFist)
	}
	isMace := func(weapon *core.Item) bool {
		return weapon != nil && weapon.WeaponType == proto.WeaponType_WeaponTypeMace
	}

	mh, oh := rogue.MainHand(), rogue.OffHand()

	if isMace(mh) || isMace(oh) {
		// "Your attacks ignore 15% of your target's armor" at rank 5: a
		// share of the target's armor (PseudoStats.ArmorIgnorePercent, as
		// the warrior's Weaponmaster mace clause), not flat armor
		// penetration. It is per attacker, so one mace in either hand
		// grants it.
		rogue.PseudoStats.ArmorIgnorePercent += armorIgnore
	}

	// The client row (13960) is a server dummy; its text gives the crit
	// to "your melee weapon attacks", so it is melee-table crit and stays
	// off the poisons and weapon procs, which roll spell crit.
	if isDaggerOrFist(mh) {
		rogue.AddStat(stats.MeleeCrit, critBonus)
		if !isDaggerOrFist(oh) {
			// MH grant above also raised the (shared) OH crit chance; the
			// default character pane shows MH's crit only, same
			// correction applyWeaponSpecializations' commented Dagger/
			// Fist Specialization already made.
			rogue.OnSpellRegistered(func(spell *core.Spell) {
				if isOffHandStrike(spell) {
					spell.BonusCritRating -= critBonus
				}
			})
		}
	} else if isDaggerOrFist(oh) {
		rogue.OnSpellRegistered(func(spell *core.Spell) {
			if isOffHandStrike(spell) {
				spell.BonusCritRating += critBonus
			}
		})
	}

	if isAxeOrSword(mh) || isAxeOrSword(oh) {
		icd := core.Cooldown{
			Timer:    rogue.NewTimer(),
			Duration: 200 * time.Millisecond,
		}

		rogue.RegisterAura(core.Aura{
			Label:    "Hack and Slash",
			ActionID: actionID,
			Duration: core.NeverExpires,
			OnReset: func(aura *core.Aura, sim *core.Simulation) {
				aura.Activate(sim)
			},
			OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if !result.Landed() || !icd.IsReady(sim) {
					return
				}

				if isAxeOrSword(mh) && spell.ProcMask.Matches(core.ProcMaskMeleeMH) && sim.Proc(extraAttackChance, "Hack and Slash (MH)") {
					icd.Use(sim)
					rogue.AutoAttacks.ExtraMHAttack(sim, 1, actionID, spell.ActionID)
					return
				}
				if isAxeOrSword(oh) && spell.ProcMask.Matches(core.ProcMaskMeleeOH) && sim.Proc(extraAttackChance, "Hack and Slash (OH)") {
					icd.Use(sim)
					rogue.AutoAttacks.ExtraOHAttack(sim, 1, actionID, spell.ActionID)
				}
			},
		})
	}
}

// applyFlawlessExecution discounts Eviscerate's Energy cost. Deferred to
// Env.RegisterPreFinalizeEffect for the same reason applyPuncturingWounds
// is: rogue.Eviscerate is nil until Initialize runs, which is after
// ApplyTalents.
func (rogue *Rogue) applyFlawlessExecution() {
	if !rogue.Talents.FlawlessExecution {
		return
	}

	rogue.Env.RegisterPreFinalizeEffect(func() {
		if rogue.Eviscerate != nil {
			rogue.Eviscerate.Cost.FlatModifier -= flawlessExecutionEnergyCostReduction
		}
	})
}

// efficientStrikesCostReduction is Efficient Strikes' rank -> Energy off
// Sinister Strike and Hemorrhage, index 0 unused: 13732 e0 aura 107 (cost,
// misc 14) on mask 33554434 (Sinister Strike 2 + Hemorrhage 33554432), the
// trait curve's -3/-5 (TraitDefinitionEffectPoints 23394, curve 83038).
var efficientStrikesCostReduction = [3]int32{0, 3, 5}

// applyEfficientStrikes discounts Sinister Strike and Hemorrhage, deferred
// for the same reason applyFlawlessExecution is.
func (rogue *Rogue) applyEfficientStrikes() {
	if rogue.Talents.EfficientStrikes == 0 {
		return
	}

	reduction := efficientStrikesCostReduction[rankIndex(rogue.Talents.EfficientStrikes, efficientStrikesCostReduction[:])]
	rogue.Env.RegisterPreFinalizeEffect(func() {
		for _, spell := range []*core.Spell{rogue.SinisterStrike, rogue.Hemorrhage} {
			if spell != nil {
				spell.Cost.FlatModifier -= reduction
			}
		}
	})
}

// applySetup adds a combo point against the attacker when the rogue
// dodges a melee attack or fully resists a spell, the same
// OnSpellHitTaken shape riposte.go's OutcomeParry listener already uses.
// Patchwerk-style single-target DPS testing never has the boss attack a
// non-tanking rogue, so this is inert there in practice; it is
// implemented anyway because the mechanic itself is a straightforward,
// testable combo-point source whenever the precondition occurs (e.g. an
// off-tanking rogue, or a cleave/AoE encounter), the same good-faith
// treatment riposte.go already gets.
func (rogue *Rogue) applySetup() {
	rank := rogue.Talents.Setup
	if rank == 0 {
		return
	}

	procChance := setupComboPointChance[rankIndex(rank, setupComboPointChance[:])]
	cpMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 13983})

	rogue.RegisterAura(core.Aura{
		Label:    "Setup",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			dodged := result.Outcome.Matches(core.OutcomeDodge)
			fullyResistedSpell := spell.DefenseType == core.DefenseTypeMagic && result.Outcome.Matches(core.OutcomeMiss)
			if !dodged && !fullyResistedSpell {
				return
			}
			if sim.Proc(procChance, "Setup") {
				rogue.AddComboPoints(sim, 1, spell.Unit, cpMetrics)
			}
		},
	})
}

// rankIndex clamps a talent rank to a lookup table's highest index.
// core.FillTalentsProto does not validate a talent string against the
// client's max rank per node, so a string with more points in a talent
// than the talent allows (or a corrupt/hand-edited one) would otherwise
// index one of the tables above out of range instead of reading the
// talent's max-rank value. Same rationale as mage/talents.go's rankIndex.
func rankIndex[T any](rank int32, table []T) int {
	if i := int(rank); i >= 0 && i < len(table) {
		return i
	}
	return len(table) - 1
}
