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
	rogue.applyMurder()
	rogue.applyRelentlessStrikes()
	rogue.applySealFate()
	rogue.applyWeaponSpecializations()
	rogue.applyWeaponExpertise()
	rogue.applyInitiative()
	rogue.applyPuncturingWounds()
	rogue.applyHackAndSlash()
	rogue.applyCutthroat()
	rogue.applyThousandCuts()
	rogue.applyQuietus()
	rogue.applyFlawlessExecution()
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

	// Master of Deception, Improved Distract and Heightened Senses are
	// all Stealth-detection or being-hit-by-ranged/spells mitigation:
	// nothing in this sim models an enemy detecting a stealthed rogue
	// or casting at one, so none of the three has a number to change.
	_, _, _ = rogue.Talents.MasterOfDeception, rogue.Talents.ImprovedDistract, rogue.Talents.HeightenedSenses

	// Dirty Tricks discounts Sap and Blind. Neither has a spell file in
	// this package: both are pre-pull/CC utility, not a Patchwerk
	// rotation ability.
	_ = rogue.Talents.DirtyTricks

	rogue.AddStat(stats.Dodge, 1*float64(rogue.Talents.LightningReflexes))
	rogue.AddStat(stats.Parry, 1*float64(rogue.Talents.Deflection))
	rogue.AddStat(stats.Crit, 1*float64(rogue.Talents.Malice))
	rogue.AddStat(stats.Hit, 1*float64(rogue.Talents.Precision))
	// TODO: Test the Armor reduction amount
	rogue.AddStat(stats.ArmorPenetration, float64(5/3*rogue.Talents.SerratedBlades*rogue.Level))
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

// Murder talent
func (rogue *Rogue) applyMurder() {
	if rogue.Talents.Murder == 0 {
		return
	}

	// post finalize, since attack tables need to be setup
	rogue.Env.RegisterPostFinalizeEffect(func() {
		for _, t := range rogue.Env.Encounter.Targets {
			switch t.MobType {
			case proto.MobType_MobTypeHumanoid, proto.MobType_MobTypeGiant, proto.MobType_MobTypeBeast, proto.MobType_MobTypeDragonkin:
				multiplier := []float64{1, 1.01, 1.02}[rogue.Talents.Murder]
				for _, at := range rogue.AttackTables[t.UnitIndex] {
					at.DamageDealtMultiplier *= multiplier
					at.CritMultiplier *= multiplier
				}
			}
		}
	})
}

func (rogue *Rogue) applyRelentlessStrikes() {
	if !rogue.Talents.RelentlessStrikes {
		return
	}

	cpMetrics := rogue.NewEnergyMetrics(core.ActionID{SpellID: 14179})
	rogue.OnComboPointsSpent(func(sim *core.Simulation, spell *core.Spell, comboPoints int32) {
		if sim.Proc(0.2*float64(comboPoints), "RelentlessStrikes") {
			rogue.AddEnergy(sim, 25, cpMetrics)
		}
	})
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

// Seal Fate talent
func (rogue *Rogue) applySealFate() {
	if rogue.Talents.SealFate == 0 {
		return
	}

	procChance := 0.2 * float64(rogue.Talents.SealFate)
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
			if spell == rogue.Garrote || spell == rogue.Ambush {
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
		SpellCode:     SpellCode_RogueBladeFlurry,
		ActionID:      core.ActionID{SpellID: 13877},
		Flags:         core.SpellFlagAPL,
		RequiredLevel: 1,

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
			if sim.GetRemainingDuration() > cooldownDur+time.Second*15 {
				// We'll have enough time to cast another BF, so use it immediately to make sure we get the 2nd one.
				return true
			}

			// Since this is our last BF, wait until we have SND / procs up.
			sndTimeRemaining := rogue.SliceAndDiceAura.RemainingDuration(sim)
			return sndTimeRemaining >= time.Second
		},
	})
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
			rogue.ApplyEnergyTickMultiplier(1.0)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.ApplyEnergyTickMultiplier(-1.0)
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
	// Cutthroat: "Your Backstab has a 3%/6%/9%/12%/15% chance to cause
	// your next Ambush within 10 sec to not require Stealth."
	cutthroatProcChancePerRank = 0.03
	cutthroatDuration          = 10 * time.Second
	// Thousand Cuts (1 rank only): "When your Rupture ability deals
	// periodic damage, the Energy cost of your next Hemorrhage or
	// Backstab ability within 10 sec is reduced by 3, stacking up to 5
	// times."
	thousandCutsCostReductionPerStack = 3
	thousandCutsMaxStacks             = 5
	thousandCutsDuration              = 10 * time.Second
	// Quietus: "Your Sinister Strike, Ghostly Strike, and Hemorrhage
	// abilities cause 2%/4%/6%/8%/10% more damage against targets below
	// 35% health." sim.IsExecutePhase35 is the same "target below 35%
	// health" the warrior Execute/priest Shadow Word: Death talents key
	// off, since this engine has no real target HP pool to read.
	quietusDamagePerRank = 0.02
	// Flawless Execution (1 rank only): "Reduces the Energy cost of your
	// Eviscerate ability by 10."
	flawlessExecutionEnergyCostReduction = 10
	// Setup: "Gives you a 33%/67%/100% chance to add a Combo Point to
	// your target after Dodging an attack or fully resisting a spell."
	// Not an exact 33.33%/rank, so a table rather than a multiply.
)

// setupComboPointChance is Setup's rank -> proc chance, index 0 unused.
var setupComboPointChance = [4]float64{0, 0.33, 0.67, 1.0}

// applyPuncturingWounds gives Backstab and Mutilate's crit bonus and
// Backstab's extra-combo-point chance. The crit bonus is applied once,
// deferred to Env.RegisterPreFinalizeEffect: ApplyTalents runs before
// Initialize (core/character.go's applyAllEffects calls agent.ApplyTalents
// first), so rogue.Backstab/MutilateMH/MutilateOH are all still nil here;
// by the pre-finalize phase every player has finished Initialize (same
// reasoning as applyMurder's RegisterPostFinalizeEffect below, which needs
// AttackTables instead of spells).
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
	armorPenRating := hackAndSlashArmorPenPctPerRank * float64(rank) * core.ArmorPenPerPercentArmor
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
		// No per-weapon armor-pen stat exists (stats.ArmorPenetration is
		// unit-wide), so one mace in either hand grants the full bonus,
		// the same simplification applyWeaponSpecializations' commented
		// Mace Specialization used for weapon skill.
		rogue.AddStat(stats.ArmorPenetration, armorPenRating)
	}

	if isDaggerOrFist(mh) {
		rogue.AddStat(stats.Crit, critBonus)
		if !isDaggerOrFist(oh) {
			// MH grant above also raised the (shared) OH crit chance; the
			// default character pane shows MH's crit only, same
			// correction applyWeaponSpecializations' commented Dagger/
			// Fist Specialization already made.
			rogue.OnSpellRegistered(func(spell *core.Spell) {
				if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
					spell.BonusCritRating -= critBonus
				}
			})
		}
	} else if isDaggerOrFist(oh) {
		rogue.OnSpellRegistered(func(spell *core.Spell) {
			if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
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

// applyCutthroat registers the Ambush-stealth-bypass buff; ambush.go
// reads rogue.CutthroatAura.
func (rogue *Rogue) applyCutthroat() {
	rank := rogue.Talents.Cutthroat
	if rank == 0 {
		return
	}

	procChance := cutthroatProcChancePerRank * float64(rank)

	rogue.CutthroatAura = rogue.RegisterAura(core.Aura{
		Label:    "Cutthroat",
		ActionID: core.ActionID{SpellID: 462708},
		Duration: cutthroatDuration,
	})

	rogue.RegisterAura(core.Aura{
		Label:    "Cutthroat Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell != rogue.Backstab || !result.Landed() {
				return
			}
			if sim.Proc(procChance, "Cutthroat") {
				rogue.CutthroatAura.Activate(sim)
			}
		},
	})
}

// applyThousandCuts stacks an Energy-cost discount on Hemorrhage/Backstab
// off Rupture ticks. The discount aura's own stack count is the only
// state; see its OnReset for why the aura unwinds its own cost delta
// before re-registering each iteration, rather than relying on Deactivate
// (which does not reset stacks - see aura.go's SetStacks/Deactivate).
func (rogue *Rogue) applyThousandCuts() {
	if !rogue.Talents.ThousandCuts {
		return
	}

	affectedSpells := func() []*core.Spell {
		spells := make([]*core.Spell, 0, 2)
		if rogue.Hemorrhage != nil {
			spells = append(spells, rogue.Hemorrhage)
		}
		if rogue.Backstab != nil {
			spells = append(spells, rogue.Backstab)
		}
		return spells
	}

	var discountAura *core.Aura
	discountAura = rogue.RegisterAura(core.Aura{
		Label:     "Thousand Cuts",
		ActionID:  core.ActionID{SpellID: 1310721},
		Duration:  thousandCutsDuration,
		MaxStacks: thousandCutsMaxStacks,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
			delta := int32(thousandCutsCostReductionPerStack) * (newStacks - oldStacks)
			for _, spell := range affectedSpells() {
				spell.Cost.FlatModifier -= delta
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			// Natural 10-sec expiry without a consuming cast: unwind via
			// SetStacks(0), not Deactivate, so OnStacksChange fires and
			// reverses the FlatModifier. Deactivate is already mid-call
			// on this aura (that is how OnExpire was reached), and it
			// guards re-entry with its own aura.active check, so the
			// reentrant Deactivate inside SetStacks(0) is a safe no-op.
			aura.SetStacks(sim, 0)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell != rogue.Hemorrhage && spell != rogue.Backstab {
				return
			}
			aura.SetStacks(sim, 0)
		},
	})

	rogue.RegisterAura(core.Aura{
		Label:    "Thousand Cuts Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnPeriodicDamageDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell != rogue.Rupture {
				return
			}
			discountAura.Activate(sim)
			discountAura.AddStack(sim)
		},
	})
}

// applyQuietus toggles Sinister Strike/Ghostly Strike/Hemorrhage's
// damage bonus on sim.IsExecutePhase35, the same "target below 35%
// health" proxy warrior/execute.go and priest/shadow_word_death.go read
// (this engine has no real target HP pool). RegisterExecutePhaseCallback
// is re-registered from OnReset because sim.reset (core/sim.go) clears
// the whole callback list at the start of every iteration; the `applied`
// flag undoes a previous iteration's still-active bonus before that
// iteration's fresh registration, since nothing else reverses it when a
// fight simply ends while still under 35%.
func (rogue *Rogue) applyQuietus() {
	rank := rogue.Talents.Quietus
	if rank == 0 {
		return
	}

	damageBonus := quietusDamagePerRank * float64(rank)
	applied := false

	affectedSpells := func() []*core.Spell {
		spells := make([]*core.Spell, 0, 3)
		for _, spell := range []*core.Spell{rogue.SinisterStrike, rogue.GhostlyStrike, rogue.Hemorrhage} {
			if spell != nil {
				spells = append(spells, spell)
			}
		}
		return spells
	}

	rogue.RegisterAura(core.Aura{
		Label:    "Quietus",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)

			if applied {
				for _, spell := range affectedSpells() {
					spell.DamageMultiplierAdditive -= damageBonus
				}
				applied = false
			}

			sim.RegisterExecutePhaseCallback(func(sim *core.Simulation, isExecute int32) {
				if isExecute > 35 || applied {
					return
				}
				applied = true
				for _, spell := range affectedSpells() {
					spell.DamageMultiplierAdditive += damageBonus
				}
			})
		},
	})
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
