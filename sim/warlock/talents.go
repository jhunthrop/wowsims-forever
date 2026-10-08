package warlock

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

func (warlock *Warlock) ApplyTalents() {
	warlock.applyWeaponImbue()

	// Affliction
	warlock.applySuppression()
	warlock.applyNightfall()
	warlock.applyShadowMastery()
	warlock.applyMalediction()
	warlock.applyImprovedDrains()
	warlock.applyImprovedBaneOfAgony()
	warlock.applyFelConcentration()
	warlock.applyPandemic()
	warlock.applyMalevolence()
	warlock.applySoulSiphon()

	// Soul Harvest (2 ranks; "Soul Harvesting" before the live tree renamed it): "You gain Soul Harvest for 10 sec if a
	// victim is killed while afflicted with your Drain Soul... 50%/100%
	// increase to your Mana regeneration." This sim has no mid-encounter
	// kill event to trigger from: the only target in a DPS test suite is
	// the boss, and it "dies" only when the encounter's fixed duration
	// ends, by which point there is no more casting left to regen mana
	// for. The talent is a no-op against every target this engine ever
	// simulates, not merely unmodelled.
	_ = warlock.Talents.SoulHarvest

	// Curse of Exhaustion (1 rank) grants a pure movement-speed snare
	// with no damage, hit, crit or resource component; this package
	// registers no spell for it (the engine has no caster-chases-target
	// movement model for a snare to matter against).
	_ = warlock.Talents.CurseOfExhaustion

	// Demonology
	warlock.applyDemonicEmbrace()
	warlock.applyFelIntellect()
	warlock.registerFelDominationCD()
	warlock.applyFelStamina()
	warlock.applyMasterSummoner()
	warlock.applyMasterDemonologist()
	warlock.applyDemonicSacrifice()
	warlock.applySoulLink()
	warlock.applyFelVitality()
	warlock.applyDemonicEnergies()
	warlock.applyDemonicKnowledge()

	// Improved Health Funnel (2 ranks) only improves Health Funnel
	// (amount, health cost, threat); this package registers no Health
	// Funnel spell at all, so there is nothing for it to improve.
	_ = warlock.Talents.ImprovedHealthFunnel

	// Demonic Aegis (2 ranks) improves Demon Skin/Demon Armor's armor
	// and resistance. Both are purely defensive (damage TAKEN, not
	// dealt) and never change a DPS number.
	_ = warlock.Talents.DemonicAegis

	// Improved Voidwalker (3 ranks) improves Torment, Consume Shadows,
	// Sacrifice and Suffering - none of which this package registers as
	// a castable ability (voidwalker.go gives the Voidwalker only auto
	// attacks). The Voidwalker is Demonology's tank/utility pet here,
	// not a DPS pet, so none of this talent's targets exist to improve.
	_ = warlock.Talents.ImprovedVoidwalker

	// Improved Felhunter (3 ranks) improves Tainted Blood, Devour Magic
	// and Paranoia and shortens Spell Lock's cooldown - none of which
	// this package registers (felhunter.go gives the Felhunter only
	// auto attacks). Same reasoning as Improved Voidwalker: the
	// Felhunter is a utility pet here, not a DPS pet.
	_ = warlock.Talents.ImprovedFelhunter

	// Demonic Brand (3 ranks): the threat-reduction half ("Searing Pain
	// generates 17/33/50% less threat") is not a DPS stat. The other
	// half - the pet's next 2/4/6 attacks dealing bonus Fire or Shadow
	// damage "based on the pet" - has no resolvable number anywhere in
	// the client data this fork reads: talents/warlock.json's
	// description carries the unresolved template variables $<minDam>
	// and $<maxDam>, and spellconst/warlock.json's single entry for
	// spell 1293695 (shared, unchanged, across all three ranks) carries
	// only the threat-reduction effect (aura 108) and a second effect
	// (aura 107, amount 3) that is neither a flat damage amount nor
	// consistent with the tooltip's rank-3 "6 attacks" - inventing a
	// number here would not be a client number.
	_ = warlock.Talents.DemonicBrand

	// Destruction
	warlock.applyImprovedShadowBolt()
	warlock.applyCataclysm()
	warlock.applyBane()
	warlock.applyDevastation()
	warlock.applyRuin()
	warlock.applyEmberstorm()
	warlock.applyAftermath()
	warlock.applyIntensity()
	warlock.applyAgonizingFlames()
	warlock.applyBaneOfHavoc()
	warlock.applyFireAndBrimstone()
	warlock.applyShadowAndFlame()

	// Destructive Reach (2 ranks) is +10/20% spell range; this engine
	// has no range/positioning model for any class, so nothing reads it.
	_ = warlock.Talents.DestructiveReach

	// Molten Skin (5 ranks) is -2/4/6/8/10% damage TAKEN: purely
	// defensive, never a DPS number.
	_ = warlock.Talents.MoltenSkin

	// Pyroclasm (2 ranks) gives Soul Fire, Rain of Fire and Hellfire a
	// 13/26% chance to Stun. This engine has no stun/CC primitive at
	// all (core/racials.go's own comment on a stun-removal trinket:
	// "no combat effect" - stuns are not modelled on either side), and
	// Hellfire itself is not registered in this package, so there is
	// nothing this talent's proc could change even if it fired.
	_ = warlock.Talents.Pyroclasm
}

func (warlock *Warlock) applyWeaponImbue() {
	if warlock.GetCharacter().Equipment.OffHand().Type != proto.ItemType_ItemTypeUnknown {
		return
	}

	level := warlock.Level
	if warlock.Options.WeaponImbue == proto.WarlockOptions_Firestone {
		warlock.applyFirestone()
	}
	if warlock.Options.WeaponImbue == proto.WarlockOptions_Spellstone {
		if level >= 55 {
			warlock.AddStat(stats.Crit, 1*core.CritRatingPerCritChance)
		}
	}
}

func (warlock *Warlock) applyFirestone() {
	level := warlock.Level

	damageMin := 0.0
	damageMax := 0.0

	// TODO: Test for spell scaling
	spellCoeff := 0.0
	spellId := int32(0)

	// TODO: Test PPM
	ppm := warlock.AutoAttacks.NewPPMManager(8, core.ProcMaskMelee)

	// FOREVER: Improved Firestone is not in the client's trees.
	// firestoneMulti := 1.0 + float64(warlock.Talents.ImprovedFirestone)*0.15
	firestoneMulti := 1.0

	if level >= 56 {
		warlock.AddStat(stats.FirePower, 21*firestoneMulti)
		damageMin = 80.0
		damageMax = 120.0
		spellId = 17949
	} else if level >= 46 {
		warlock.AddStat(stats.FirePower, 17*firestoneMulti)
		damageMin = 60.0
		damageMax = 90.0
		spellId = 17947
	} else if level >= 36 {
		warlock.AddStat(stats.FirePower, 14*firestoneMulti)
		damageMin = 40.0
		damageMax = 60.0
		spellId = 17945
	} else if level >= 28 {
		warlock.AddStat(stats.FirePower, 10*firestoneMulti)
		damageMin = 25.0
		damageMax = 35.0
		spellId = 758
	}

	if level >= 28 && warlock.Consumes.MainHandImbue == proto.WeaponImbue_WeaponImbueUnknown {
		fireProcSpell := warlock.GetOrRegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: spellId},
			SpellSchool: core.SpellSchoolFire,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskEmpty,

			DamageMultiplier:         firestoneMulti,
			ThreatMultiplier:         1,
			DamageMultiplierAdditive: 1,
			BonusCoefficient:         spellCoeff,

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				baseDamage := sim.Roll(damageMin, damageMax)

				spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicCrit)
			},
		})

		core.MakePermanent(warlock.GetOrRegisterAura(core.Aura{
			Label: "Firestone Proc",
			OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if !result.Landed() {
					return
				}

				if !spell.ProcMask.Matches(core.ProcMaskMelee) {
					return
				}

				if !ppm.Proc(sim, core.ProcMaskMelee, "Firestone Proc") {
					return
				}

				fireProcSpell.Cast(sim, result.Target)
			},
		}))
	}
}

///////////////////////////////////////////////////////////////////////////
//                            Affliction
///////////////////////////////////////////////////////////////////////////

const (
	// Suppression (18174, live tree): "Improves your chance to hit by N%
	// and reduces all threat you generate by 4N%". The client's rows at
	// rank 5 are aura 55 (spell hit) 5 and aura 54 (hit) 5, both with
	// spell class mask 0 (every spell), and aura 10 (threat) -20 on misc
	// 127 (all schools).
	suppressionHitPercentPerRank      = 1
	suppressionThreatReductionPerRank = 0.04
)

func (warlock *Warlock) applySuppression() {
	if warlock.Talents.Suppression == 0 {
		return
	}

	points := float64(warlock.Talents.Suppression)
	warlock.AddStat(stats.Hit, suppressionHitPercentPerRank*points*core.HitRatingPerHitChance)
	warlock.PseudoStats.ThreatMultiplier *= 1 - suppressionThreatReductionPerRank*points
}

func (warlock *Warlock) applyNightfall() {
	if warlock.Talents.Nightfall <= 0 {
		return
	}

	shadowTranceAura := warlock.RegisterAura(core.Aura{
		Label:    "Nightfall Shadow Trance",
		ActionID: core.ActionID{SpellID: 17941},
		Duration: time.Second * 10,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.ShadowBolt {
				spell.CastTimeMultiplier -= 1
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.ShadowBolt {
				spell.CastTimeMultiplier += 1
			}
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			// Check if the shadowbolt was instant cast and not a normal one
			if spell.SpellCode == SpellCode_WarlockShadowBolt && spell.CurCast.CastTime == 0 {
				aura.Deactivate(sim)
			}
		},
	})

	procChance := 0.02 * float64(warlock.Talents.Nightfall)

	core.MakePermanent(warlock.RegisterAura(core.Aura{
		Label: "Nightfall Hidden Aura",
		OnPeriodicDamageDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if (spell.SpellCode == SpellCode_WarlockCorruption || spell.SpellCode == SpellCode_WarlockDrainLife) && sim.Proc(procChance, "Nightfall") {
				shadowTranceAura.Activate(sim)
			}
		},
	}))
}

func (warlock *Warlock) applyShadowMastery() {
	if warlock.Talents.ShadowMastery == 0 {
		return
	}

	// These spells have their base damage modded instead
	// Apply Aura: Modifies Spell Effectiveness (8)
	excludedSpellCodes := []int32{SpellCode_WarlockCurseOfAgony, SpellCode_WarlockDeathCoil, SpellCode_WarlockDrainLife, SpellCode_WarlockDrainSoul}

	warlock.OnSpellRegistered(func(spell *core.Spell) {
		// Shadow Mastery applies a base damage modifier to all dots / channeled spells instead
		if spell.SpellSchool.Matches(core.SpellSchoolShadow) && isWarlockSpell(spell) && !slices.Contains(excludedSpellCodes, spell.SpellCode) {
			spell.DamageMultiplierAdditive += warlock.shadowMasteryBonus()
		}
	})
}

func (warlock *Warlock) shadowMasteryBonus() float64 {
	return .02 * float64(warlock.Talents.ShadowMastery)
}

///////////////////////////////////////////////////////////////////////////
//                            Demonology Talents
///////////////////////////////////////////////////////////////////////////

func (warlock *Warlock) applyDemonicEmbrace() {
	if warlock.Talents.DemonicEmbrace == 0 {
		return
	}

	points := float64(warlock.Talents.DemonicEmbrace)
	warlock.MultiplyStat(stats.Stamina, 1+.03*(points))
	warlock.MultiplyStat(stats.Spirit, 1-.01*(points))
}

func (warlock *Warlock) applyFelIntellect() {
	/*
		if warlock.Talents.FelIntellect == 0 {
			return
		}

		multiplier := 1 + 0.03*float64(warlock.Talents.FelIntellect)
		for _, pet := range warlock.BasePets {
			pet.MultiplyStat(stats.Mana, multiplier)
		}
	*/
}

func (warlock *Warlock) applyFelStamina() {
	/*
		if warlock.Talents.FelStamina == 0 {
			return
		}

		multiplier := 1 + 0.03*float64(warlock.Talents.FelStamina)
		for _, pet := range warlock.BasePets {
			pet.MultiplyStat(stats.Health, multiplier)
		}
	*/
}

func (warlock *Warlock) applyMasterSummoner() {
	if warlock.Talents.MasterSummoner == 0 {
		return
	}

	castTimeReduction := time.Second * 2 * time.Duration(warlock.Talents.MasterSummoner)
	costReduction := 20 * warlock.Talents.MasterSummoner

	// Use an aura because the summon spells aren't registered by this point
	warlock.RegisterAura(core.Aura{
		Label:    "Master Summoner Hidden Aura",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.SummonDemonSpells {
				spell.DefaultCast.CastTime -= castTimeReduction
				spell.Cost.Multiplier -= costReduction
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.SummonDemonSpells {
				spell.DefaultCast.CastTime += castTimeReduction
				spell.Cost.Multiplier += costReduction
			}
		},
	})
}

func (warlock *Warlock) applyMasterDemonologist() {
	if warlock.Talents.MasterDemonologist == 0 {
		return
	}

	points := float64(warlock.Talents.MasterDemonologist)
	damageDealtMultiplier := 1 + 0.02*points
	damageTakenMultiplier := 1 - 0.02*points
	threatMultiplier := 1 + -0.04*points
	bonusResistance := 2 * points

	impConfig := core.Aura{
		Label:    "Master Demonologist (Imp)",
		ActionID: core.ActionID{SpellID: 23825, Tag: 1},
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.ThreatMultiplier *= threatMultiplier
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.ThreatMultiplier /= threatMultiplier
		},
	}

	voidwalkerConfig := core.Aura{
		Label:    "Master Demonologist (Voidwalker)",
		ActionID: core.ActionID{SpellID: 23825, Tag: 2},
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.DamageTakenMultiplier *= damageTakenMultiplier
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.DamageTakenMultiplier /= damageTakenMultiplier
		},
	}

	succubusConfig := core.Aura{
		Label:    "Master Demonologist (Succubus)",
		ActionID: core.ActionID{SpellID: 23825, Tag: 3},
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.DamageDealtMultiplier *= damageDealtMultiplier
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.DamageDealtMultiplier /= damageDealtMultiplier
		},
	}

	felhunterConfig := core.Aura{
		Label:    "Master Demonologist (Felhunter)",
		ActionID: core.ActionID{SpellID: 23825, Tag: 4},
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddResistancesDynamic(sim, bonusResistance)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddResistancesDynamic(sim, -bonusResistance)
		},
	}

	for _, pet := range warlock.BasePets {
		pet.ApplyOnPetEnable(func(sim *core.Simulation) {
			if warlock.MasterDemonologistAura != nil {
				warlock.MasterDemonologistAura.Deactivate(sim)
			}
		})
	}

	warlockImpAura := warlock.RegisterAura(impConfig)
	impAura := warlock.Imp.RegisterAura(impConfig)
	warlock.Imp.ApplyOnPetEnable(func(sim *core.Simulation) {
		impAura.Activate(sim)
		warlock.MasterDemonologistAura = warlockImpAura
	})
	warlock.Imp.ApplyOnPetDisable(func(sim *core.Simulation, isSacrifice bool) {
		impAura.Deactivate(sim)
	})

	warlockVoidwalkerAura := warlock.RegisterAura(voidwalkerConfig)
	voidwalkerAura := warlock.Voidwalker.RegisterAura(voidwalkerConfig)
	warlock.Voidwalker.ApplyOnPetEnable(func(sim *core.Simulation) {
		voidwalkerAura.Activate(sim)
		warlock.MasterDemonologistAura = warlockVoidwalkerAura
	})
	warlock.Voidwalker.ApplyOnPetDisable(func(sim *core.Simulation, isSacrifice bool) {
		voidwalkerAura.Deactivate(sim)
	})

	warlockSuccubusAura := warlock.RegisterAura(succubusConfig)
	succubusAura := warlock.Succubus.RegisterAura(succubusConfig)
	warlock.Succubus.ApplyOnPetEnable(func(sim *core.Simulation) {
		succubusAura.Activate(sim)
		warlock.MasterDemonologistAura = warlockSuccubusAura
	})
	warlock.Succubus.ApplyOnPetDisable(func(sim *core.Simulation, isSacrifice bool) {
		succubusAura.Deactivate(sim)
	})

	warlockFelhunterAura := warlock.RegisterAura(felhunterConfig)
	felhunterAura := warlock.Felhunter.RegisterAura(felhunterConfig)
	warlock.Felhunter.ApplyOnPetEnable(func(sim *core.Simulation) {
		felhunterAura.Activate(sim)
		warlock.MasterDemonologistAura = warlockFelhunterAura
	})
	warlock.Felhunter.ApplyOnPetDisable(func(sim *core.Simulation, isSacrifice bool) {
		felhunterAura.Deactivate(sim)
	})

	for _, pet := range warlock.BasePets {
		pet.ApplyOnPetEnable(func(sim *core.Simulation) {
			warlock.MasterDemonologistAura.Activate(sim)
		})

		pet.ApplyOnPetDisable(func(sim *core.Simulation, isSacrifice bool) {
			if warlock.MasterDemonologistAura != nil {
				warlock.MasterDemonologistAura.Deactivate(sim)
				warlock.MasterDemonologistAura = nil
			}
		})
	}
}

func (warlock *Warlock) applySoulLink() {
	if !warlock.Talents.SoulLink {
		return
	}

	actionID := core.ActionID{SpellID: 19028}
	soulLinkConfig := core.Aura{
		Label:    "Soul Link Aura",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.DamageTakenMultiplier /= 1.3
			aura.Unit.PseudoStats.DamageDealtMultiplier *= 1.03
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.DamageDealtMultiplier /= 1.03
			aura.Unit.PseudoStats.DamageTakenMultiplier *= 1.3
		},
	}

	warlock.SoulLinkAura = warlock.RegisterAura(soulLinkConfig)
	for _, pet := range warlock.BasePets {
		pet.SoulLinkAura = pet.RegisterAura(soulLinkConfig)

		oldOnPetDisable := pet.OnPetDisable
		pet.OnPetDisable = func(sim *core.Simulation, isSacrifice bool) {
			oldOnPetDisable(sim, isSacrifice)
			warlock.SoulLinkAura.Deactivate(sim)
			pet.SoulLinkAura.Deactivate(sim)
		}
	}

	warlock.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagAPL,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warlock.ActivePet != nil
		},

		ManaCost: core.ManaCostOptions{
			BaseCost: 0.2,
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			warlock.SoulLinkAura.Activate(sim)
			warlock.ActivePet.SoulLinkAura.Activate(sim)
		},
	})
}

func (warlock *Warlock) applyDemonicSacrifice() {
	if !warlock.Talents.DemonicSacrifice {
		return
	}

	impAura := warlock.GetOrRegisterAura(core.Aura{
		Label:    "Burning Wish",
		ActionID: core.ActionID{SpellID: 18789},
		Duration: 30 * time.Minute,

		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire] *= 1.15
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire] /= 1.15
		},
	})

	var vwPa *core.PendingAction
	healthMetric := warlock.NewHealthMetrics(core.ActionID{SpellID: 18790})
	voidwalkerAura := warlock.GetOrRegisterAura(core.Aura{
		Label:    "Fel Stamina",
		ActionID: core.ActionID{SpellID: 18790},
		Duration: 30 * time.Minute,

		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			vwPa = core.NewPeriodicAction(sim, core.PeriodicActionOptions{
				Period: time.Second * 4,
				OnAction: func(s *core.Simulation) {
					warlock.GainHealth(sim, warlock.MaxHealth()*0.03, healthMetric)
				},
			})
			sim.AddPendingAction(vwPa)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			vwPa.Cancel(sim)
		},
	})

	succubusAura := warlock.GetOrRegisterAura(core.Aura{
		Label:    "Touch of Shadow",
		ActionID: core.ActionID{SpellID: 18791},
		Duration: 30 * time.Minute,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] *= 1.15
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] /= 1.15
		},
	})

	var fhPa *core.PendingAction
	manaMetric := warlock.NewManaMetrics(core.ActionID{SpellID: 18792})
	felhunterAura := warlock.GetOrRegisterAura(core.Aura{
		Label:    "Fel Energy",
		ActionID: core.ActionID{SpellID: 18792},
		Duration: 30 * time.Minute,

		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			fhPa = core.NewPeriodicAction(sim, core.PeriodicActionOptions{
				Period: time.Second * 4,
				OnAction: func(s *core.Simulation) {
					warlock.AddMana(sim, warlock.MaxMana()*0.02, manaMetric)
				},
			})
			sim.AddPendingAction(fhPa)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			fhPa.Cancel(sim)
		},
	})

	dsAuras := []*core.Aura{felhunterAura, impAura, succubusAura, voidwalkerAura}
	for _, basePet := range warlock.BasePets {
		pet := basePet
		oldOnPetEnable := pet.OnPetEnable
		pet.OnPetEnable = func(sim *core.Simulation) {
			oldOnPetEnable(sim)

			// Demonic Pact: "no longer cancelled by summoning a
			// different Demon pet. Resummoning the sacrificed pet will
			// still cancel the effect." Without the talent, summoning
			// ANY pet cancels the sacrifice buff, which is the
			// unconditional deactivate below.
			if warlock.Talents.DemonicPact && pet != warlock.sacrificedPet {
				return
			}

			for _, dsAura := range dsAuras {
				dsAura.Deactivate(sim)
			}
			warlock.sacrificedPet = nil
		}
	}

	warlock.GetOrRegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_WarlockDemonicSacrifice,
		ActionID:    core.ActionID{SpellID: 18788},
		SpellSchool: core.SpellSchoolShadow,
		Flags:       core.SpellFlagAPL,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warlock.ActivePet != nil
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			switch warlock.ActivePet {
			case warlock.Felhunter:
				felhunterAura.Activate(sim)
			case warlock.Imp:
				impAura.Activate(sim)
			case warlock.Succubus:
				succubusAura.Activate(sim)
			case warlock.Voidwalker:
				voidwalkerAura.Activate(sim)
			}

			warlock.sacrificedPet = warlock.ActivePet
			warlock.changeActivePet(sim, nil, true)
		},
	})
}

///////////////////////////////////////////////////////////////////////////
//                            Destruction Talents
///////////////////////////////////////////////////////////////////////////

func (warlock *Warlock) applyImprovedShadowBolt() {
	if warlock.Talents.ImprovedShadowBolt == 0 {
		return
	}

	warlock.ImprovedShadowBoltAuras = warlock.NewEnemyAuraArray(func(unit *core.Unit) *core.Aura {
		return core.ImprovedShadowBoltAura(unit, warlock.Talents.ImprovedShadowBolt)
	})

	affectedSpellCodes := []int32{SpellCode_WarlockShadowBolt}
	core.MakePermanent(warlock.RegisterAura(core.Aura{
		Label: "ISB Trigger",
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.ShadowBolt {
				spell.RelatedAuras = []core.AuraArray{warlock.ImprovedShadowBoltAuras}
			}
			warlock.DebuffSpells = append(warlock.DebuffSpells, warlock.ShadowBolt...)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.Landed() && result.DidCrit() && slices.Contains(affectedSpellCodes, spell.SpellCode) {
				isbAura := warlock.ImprovedShadowBoltAuras.Get(result.Target)
				isbAura.Activate(sim)
				isbAura.SetStacks(sim, isbAura.MaxStacks)
			}
		},
	}))
}

func (warlock *Warlock) applyCataclysm() {
	if warlock.Talents.Cataclysm == 0 {
		return
	}

	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(WarlockFlagDestruction) && spell.Cost != nil {
			spell.Cost.Multiplier -= warlock.Talents.Cataclysm
		}
	})
}

func (warlock *Warlock) applyBane() {
	if warlock.Talents.Bane == 0 {
		return
	}

	points := time.Duration(warlock.Talents.Bane)
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_WarlockShadowBolt || spell.SpellCode == SpellCode_WarlockImmolate {
			spell.DefaultCast.CastTime -= time.Millisecond * 100 * points
		} else if spell.SpellCode == SpellCode_WarlockSoulFire {
			spell.DefaultCast.CastTime -= time.Millisecond * 400 * points
		}
	})
}

func (warlock *Warlock) applyDevastation() {
	/*
		if warlock.Talents.Devastation == 0 {
			return
		}

		points := float64(warlock.Talents.Devastation)
		warlock.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Flags.Matches(WarlockFlagDestruction) {
				spell.BonusCritRating += points * core.CritRatingPerCritChance
			}
		})
	*/
}

func (warlock *Warlock) improvedImmolateBonus() float64 {
	// FOREVER: Improved Immolate is not in the client's trees.
	// return 0.05 * float64(warlock.Talents.ImprovedImmolate)
	return 0
}

func (warlock *Warlock) applyRuin() {
	// FOREVER: Ruin has ranks in the client's trees, so the field is an
	// int32 now rather than a bool.
	if warlock.Talents.Ruin == 0 {
		return
	}
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(WarlockFlagDestruction) {
			spell.CritDamageBonus += 1
		}
	})
}

func (warlock *Warlock) applyEmberstorm() {
	/*
		if warlock.Talents.Emberstorm == 0 {
			return
		}

		points := float64(warlock.Talents.Emberstorm)
		warlock.OnSpellRegistered(func(spell *core.Spell) {
			if spell.SpellSchool.Matches(core.SpellSchoolFire) && isWarlockSpell(spell) {
				spell.DamageMultiplierAdditive += 0.02 * points
			}
		})
	*/
}

///////////////////////////////////////////////////////////////////////////
//          Talents read from new client proto fields (this task)
///////////////////////////////////////////////////////////////////////////

// Per-rank talent values, read off the client's own rank descriptions
// for build 1.60.1.70009 (data/builds/1.60.1.70009/talents/warlock.json).
const (
	// Malediction: "Increases all periodic damage done by your Warlock
	// spells by 1/2/3/4/5%."
	maledictionPeriodicDamagePerRank = 0.01

	// Fel Concentration / Intensity share the same four values: "Gives
	// you a 23/47/70% chance to avoid interruption caused by damage
	// while channeling or casting" their respective spell lists.
	felConcentrationIntensityRank1 = 0.23
	felConcentrationIntensityRank2 = 0.47
	felConcentrationIntensityRank3 = 0.70

	// Improved Bane of Agony: "Increases the damage done by your Bane
	// of Agony by 5/10%." ("Bane of Agony" is this client build's name
	// for Curse of Agony, SpellCode_WarlockCurseOfAgony.)
	improvedBaneOfAgonyDamagePerRank = 0.05

	// Fel Vitality: "Increases the maximum health and Mana of your
	// Imp, Voidwalker, Succubus, Incubus, and Felhunter by 5/10/15%,
	// and increases your maximum Mana by 5/10/15%."
	felVitalityPerRank = 0.05

	// Agonizing Flames: "Increases the critical strike chance of your
	// Searing Pain spell by 3/7/10% and the damage done by all your
	// Destruction spells by 3/7/10%." Both halves share one table.

	// Fire and Brimstone: "Increases the critical strike chance of
	// your Conflagrate spell by 8/17/25%."

	// Shadow and Flame: "...increases all Shadow/Fire damage you deal
	// by 2/4/6/8/10% for 20 sec... Conflagrate has a 20/40/60/80/100%
	// chance not to consume Immolate..."
	shadowAndFlameDamagePerRank          = 0.02
	shadowAndFlameImmolatePreserveChance = 0.20
	shadowAndFlameBuffDuration           = 20 * time.Second

	// Bane of Havoc: "...causing 15% of all damage done by the Warlock
	// to other targets to also be dealt to the cursed target."
	baneOfHavocSpilloverPct = 0.15
	baneOfHavocDuration     = 5 * time.Minute
)

// improvedDrainsBonus, pandemicCritDamageBonus, agonizingFlamesBonus and
// fireAndBrimstoneCritPerRank are non-linear across ranks, so they read
// the client's own table rather than a multiplication.
var (
	// Improved Drains: "Increases health drained or damage done by
	// your Drain Life, Drain Soul, and Wrack spells by 7/13/20%."
	improvedDrainsBonus = [4]float64{0, 0.07, 0.13, 0.20}

	// Pandemic: "Increases the critical strike damage bonus of your
	// Corruption, Bane of Agony, Bane of Doom, Drain Soul, Drain Life,
	// Siphon Life, and Wrack spells by 33/67/100%."
	pandemicCritDamageBonus = [4]float64{0, 0.33, 0.67, 1.00}

	// Soul Siphon: "...by 4/8/12% per each of your other Affliction
	// effects active on the target, up to a maximum increase of
	// 12/24/36%" - i.e. the per-effect rate times a 3-effect cap.
	soulSiphonPerEffectBonus = [4]float64{0, 0.04, 0.08, 0.12}

	// Agonizing Flames' crit (Searing Pain) and damage (all
	// Destruction spells) halves share one percentage table.
	agonizingFlamesBonus = [4]float64{0, 0.03, 0.07, 0.10}

	// Fire and Brimstone: Conflagrate crit chance, in percentage
	// points (fed to CritRatingPerCritChance, not a fraction).
	fireAndBrimstoneCritPerRank = [4]float64{0, 8, 17, 25}

	// Demonic Knowledge: "...by up to 33/67/100% of your level..."
	demonicKnowledgeLevelPct = [4]float64{0, 0.33, 0.67, 1.00}

	// Demonic Energies' Life Tap mana share to the active pet: "...your
	// summoned demon gains 50%/100% of the Mana you gain."
	demonicEnergiesManaShare = [3]float64{0, 0.50, 1.00}
)

// soulSiphonAffectedSpellCodes is Soul Siphon's own damage/heal list:
// Drain Life, Drain Soul and Wrack.
var soulSiphonAffectedSpellCodes = []int32{SpellCode_WarlockDrainLife, SpellCode_WarlockDrainSoul, SpellCode_WarlockWrack}

// improvedDrainsAffectedSpellCodes is Improved Drains' own list, the
// same three spells Soul Siphon buffs.
var improvedDrainsAffectedSpellCodes = []int32{SpellCode_WarlockDrainLife, SpellCode_WarlockDrainSoul, SpellCode_WarlockWrack}

// pandemicAffectedSpellCodes is Pandemic's own list: Corruption, Bane
// of Agony (= Curse of Agony here), Bane of Doom (= Curse of Doom),
// Drain Soul, Drain Life, Siphon Life and Wrack.
var pandemicAffectedSpellCodes = []int32{
	SpellCode_WarlockCorruption,
	SpellCode_WarlockCurseOfAgony,
	SpellCode_WarlockCurseOfDoom,
	SpellCode_WarlockDrainSoul,
	SpellCode_WarlockDrainLife,
	SpellCode_WarlockSiphonLife,
	SpellCode_WarlockWrack,
}

// felConcentrationAffectedSpellCodes is Fel Concentration's own list:
// Drain Life, Drain Soul and Wrack. ("Drain Mana" in the tooltip has no
// spell file in this package.)
var felConcentrationAffectedSpellCodes = []int32{SpellCode_WarlockDrainLife, SpellCode_WarlockDrainSoul, SpellCode_WarlockWrack}

func (warlock *Warlock) applyMalediction() {
	rank := warlock.Talents.Malediction
	if rank == 0 {
		return
	}

	bonus := maledictionPeriodicDamagePerRank * float64(rank)
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if isWarlockSpell(spell) {
			spell.PeriodicDamageMultiplierAdditive += bonus
		}
	})
}

func (warlock *Warlock) applyImprovedDrains() {
	rank := warlock.Talents.ImprovedDrains
	if rank == 0 {
		return
	}

	bonus := improvedDrainsBonus[rank]
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if slices.Contains(improvedDrainsAffectedSpellCodes, spell.SpellCode) {
			spell.DamageMultiplierAdditive += bonus
		}
	})
}

func (warlock *Warlock) applyImprovedBaneOfAgony() {
	rank := warlock.Talents.ImprovedBaneOfAgony
	if rank == 0 {
		return
	}

	bonus := improvedBaneOfAgonyDamagePerRank * float64(rank)
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_WarlockCurseOfAgony {
			spell.DamageMultiplierAdditive += bonus
		}
	})
}

func (warlock *Warlock) felConcentrationIntensityReduction(rank int32) float64 {
	switch rank {
	case 1:
		return felConcentrationIntensityRank1
	case 2:
		return felConcentrationIntensityRank2
	case 3:
		return felConcentrationIntensityRank3
	default:
		return 0
	}
}

func (warlock *Warlock) applyFelConcentration() {
	rank := warlock.Talents.FelConcentration
	if rank == 0 {
		return
	}

	reduction := warlock.felConcentrationIntensityReduction(rank)
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if slices.Contains(felConcentrationAffectedSpellCodes, spell.SpellCode) {
			spell.PushbackReduction += reduction
		}
	})
}

func (warlock *Warlock) applyPandemic() {
	rank := warlock.Talents.Pandemic
	if rank == 0 {
		return
	}

	bonus := pandemicCritDamageBonus[rank]
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if slices.Contains(pandemicAffectedSpellCodes, spell.SpellCode) {
			spell.CritDamageBonus += bonus
		}
	})
}

func (warlock *Warlock) applyMalevolence() {
	rank := warlock.Talents.Malevolence
	if rank == 0 {
		return
	}

	bonus := float64(rank) * core.CritRatingPerCritChance
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if isWarlockSpell(spell) && spell.SpellSchool.Matches(core.SpellSchoolShadow) {
			spell.BonusCritRating += bonus
		}
	})
}

// soulSiphonMultiplier is Soul Siphon's own damage/heal multiplier for
// one snapshot of Drain Life, Drain Soul or Wrack: 1 plus the talent's
// per-effect rate, times the number of OTHER Affliction DoTs currently
// active on target (Corruption, Curse of Agony, Curse of Doom, Siphon
// Life and - for Drain Life/Drain Soul's snapshot - Wrack too), capped
// at 3 effects. excludeSpell is the caller's own spell, so Wrack (an
// Affliction DoT itself) does not count against its own snapshot.
//
// Evaluated once per OnSnapshot call (cast time for Drain Life/Drain
// Soul, since neither re-snapshots per tick; cast time for Wrack, its
// only snapshot), not continuously for the channel's duration - the
// engine's snapshot-at-cast convention Siphon Life's own OnSnapshot
// already follows for its target-modifier snapshot, and the nearest
// approximation to the tooltip's wording this architecture supports
// without a wider change to how these three spells tick.
func (warlock *Warlock) soulSiphonMultiplier(target *core.Unit, excludeSpell *core.Spell) float64 {
	rank := warlock.Talents.SoulSiphon
	if rank == 0 {
		return 1
	}

	const maxEffects = 3
	perEffect := soulSiphonPerEffectBonus[rank]

	count := 0
	for _, spell := range warlock.DoTSpells {
		if spell == excludeSpell || !spell.Flags.Matches(WarlockFlagAffliction) {
			continue
		}
		if spell.Dot(target).IsActive() {
			count++
		}
	}
	if count > maxEffects {
		count = maxEffects
	}

	return 1 + perEffect*float64(count)
}

func (warlock *Warlock) applySoulSiphon() {
	// soulSiphonMultiplier reads warlock.Talents.SoulSiphon itself and
	// is called unconditionally from drain_life.go, drain_soul.go and
	// wrack.go's own OnSnapshot hooks; there is nothing to register
	// here when the talent is untaken, but the function exists (rather
	// than inlining Talents.SoulSiphon checks at every call site) so
	// ApplyTalents reads the same way for every Affliction talent in
	// this file.
	_ = warlock.Talents.SoulSiphon
}

func (warlock *Warlock) applyFelVitality() {
	rank := warlock.Talents.FelVitality
	if rank == 0 {
		return
	}

	multiplier := 1 + felVitalityPerRank*float64(rank)
	warlock.MultiplyStat(stats.Mana, multiplier)
	for _, pet := range warlock.BasePets {
		pet.MultiplyStat(stats.Mana, multiplier)
		pet.MultiplyStat(stats.Health, multiplier)
	}
}

// shareDemonicEnergiesMana is Demonic Energies' Life Tap half: "When
// you gain Mana from Life Tap, your summoned demon gains 50%/100% of
// the Mana you gain." Called directly from lifetap.go's ApplyEffects
// with the amount of Mana the cast just restored.
//
// The talent's other half - healing the pet for 8%/15% of all spell
// damage the warlock deals - is not modelled: WarlockPet never calls
// core.Unit.EnableHealthBar (pet.go), so GainHealth panics on a nil
// health bar for every pet in this package. Pet survivability is not
// tracked here regardless, and this half of the talent does not change
// the warlock's own DPS.
func (warlock *Warlock) shareDemonicEnergiesMana(sim *core.Simulation, amount float64) {
	rank := warlock.Talents.DemonicEnergies
	if rank == 0 || warlock.ActivePet == nil {
		return
	}

	share := demonicEnergiesManaShare[rank]
	warlock.ActivePet.AddMana(sim, amount*share, warlock.ActivePet.LifeTapManaMetrics)
}

func (warlock *Warlock) applyDemonicEnergies() {
	// shareDemonicEnergiesMana reads warlock.Talents.DemonicEnergies
	// itself and is called unconditionally from lifetap.go; see that
	// function's own comment for why the talent's other half (pet
	// healing) is not modelled. Exists so ApplyTalents reads uniformly;
	// see applySoulSiphon's identical reasoning.
	_ = warlock.Talents.DemonicEnergies
}

// applyDemonicKnowledge grants Demonic Knowledge's spell-power bonus to
// the warlock and to whichever pet is currently summoned, for as long
// as any pet is active - "while you have a summoned Demon pet active"
// covers Imp, Voidwalker, Succubus and Felhunter alike here, so one
// warlock-side aura and one pet-side aura per pet, toggled from each
// pet's own enable/disable hooks, cover all four.
func (warlock *Warlock) applyDemonicKnowledge() {
	rank := warlock.Talents.DemonicKnowledge
	if rank == 0 {
		return
	}

	bonus := demonicKnowledgeLevelPct[rank] * float64(warlock.Level)

	warlockAura := warlock.RegisterAura(core.Aura{
		Label:    "Demonic Knowledge",
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warlock.AddStatDynamic(sim, stats.SpellPower, bonus)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warlock.AddStatDynamic(sim, stats.SpellPower, -bonus)
		},
	})

	for _, basePet := range warlock.BasePets {
		pet := basePet
		petAura := pet.RegisterAura(core.Aura{
			Label:    "Demonic Knowledge (Pet)",
			Duration: core.NeverExpires,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				pet.AddStatDynamic(sim, stats.SpellPower, bonus)
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				pet.AddStatDynamic(sim, stats.SpellPower, -bonus)
			},
		})

		pet.ApplyOnPetEnable(func(sim *core.Simulation) {
			warlockAura.Activate(sim)
			petAura.Activate(sim)
		})
		pet.ApplyOnPetDisable(func(sim *core.Simulation, isSacrifice bool) {
			warlockAura.Deactivate(sim)
			petAura.Deactivate(sim)
		})
	}
}

// aftermathInitialDamageBonus is Aftermath's Immolate half: "Increases
// the initial damage of your Immolate spell by 10/20/30/40/50%."
// immolate.go's own ApplyEffects reads this into the same temporary
// DamageMultiplier slot improvedImmolateBonus (a talent absent from the
// client's trees) already uses for the initial hit only, leaving the
// periodic tick untouched.
//
// Aftermath's other half - Conflagrate gaining a 20/40/60/80/100%
// chance to Daze the target - is a movement-speed snare with no damage,
// hit, crit, cast-time or resource component, the same reasoning
// Curse of Exhaustion's marker in ApplyTalents gives.
func (warlock *Warlock) aftermathInitialDamageBonus() float64 {
	return 0.10 * float64(warlock.Talents.Aftermath)
}

func (warlock *Warlock) applyAftermath() {
	// aftermathInitialDamageBonus reads warlock.Talents.Aftermath
	// itself and is called unconditionally from immolate.go; exists so
	// ApplyTalents reads uniformly, see applySoulSiphon's reasoning.
	_ = warlock.Talents.Aftermath
}

func (warlock *Warlock) applyIntensity() {
	rank := warlock.Talents.Intensity
	if rank == 0 {
		return
	}

	reduction := warlock.felConcentrationIntensityReduction(rank)
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(WarlockFlagDestruction) {
			spell.PushbackReduction += reduction
		}
	})
}

func (warlock *Warlock) applyAgonizingFlames() {
	rank := warlock.Talents.AgonizingFlames
	if rank == 0 {
		return
	}

	bonus := agonizingFlamesBonus[rank]
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if !spell.Flags.Matches(WarlockFlagDestruction) {
			return
		}
		spell.DamageMultiplierAdditive += bonus
		if spell.SpellCode == SpellCode_WarlockSearingPain {
			spell.BonusCritRating += bonus * 100 * core.CritRatingPerCritChance
		}
	})
}

func (warlock *Warlock) applyFireAndBrimstone() {
	rank := warlock.Talents.FireAndBrimstone
	if rank == 0 {
		return
	}

	bonus := fireAndBrimstoneCritPerRank[rank] * core.CritRatingPerCritChance
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_WarlockConflagrate {
			spell.BonusCritRating += bonus
		}
	})
}

// applyShadowAndFlame registers the two 20-second self-buff auras
// Shadow and Flame grants on a landed Conflagrate (Shadow damage) or
// Shadowburn (Fire damage); conflagrate.go and shadowburn.go each call
// triggerShadowAndFlame from their own ApplyEffects on a landed hit.
// shadowAndFlamePreservesImmolate is the talent's other Conflagrate
// half, read directly from conflagrate.go before it decides whether to
// consume Immolate. The Soul Shard refund on Shadowburn is not
// modelled: see shadowburn.go's own comment.
func (warlock *Warlock) applyShadowAndFlame() {
	rank := warlock.Talents.ShadowAndFlame
	if rank == 0 {
		return
	}

	bonus := shadowAndFlameDamagePerRank * float64(rank)

	warlock.shadowAndFlameShadowAura = warlock.RegisterAura(core.Aura{
		Label:    "Shadow and Flame (Shadow)",
		ActionID: core.ActionID{SpellID: 426316, Tag: 1},
		Duration: shadowAndFlameBuffDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] *= 1 + bonus
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] /= 1 + bonus
		},
	})

	warlock.shadowAndFlameFireAura = warlock.RegisterAura(core.Aura{
		Label:    "Shadow and Flame (Fire)",
		ActionID: core.ActionID{SpellID: 426316, Tag: 2},
		Duration: shadowAndFlameBuffDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire] *= 1 + bonus
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warlock.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire] /= 1 + bonus
		},
	})
}

// triggerShadowAndFlame activates the matching Shadow and Flame buff
// for a landed Conflagrate or Shadowburn hit. No-op when the talent is
// untaken (the auras are nil).
func (warlock *Warlock) triggerShadowAndFlame(sim *core.Simulation, spellCode int32) {
	switch spellCode {
	case SpellCode_WarlockConflagrate:
		if warlock.shadowAndFlameShadowAura != nil {
			warlock.shadowAndFlameShadowAura.Activate(sim)
		}
	case SpellCode_WarlockShadowburn:
		if warlock.shadowAndFlameFireAura != nil {
			warlock.shadowAndFlameFireAura.Activate(sim)
		}
	}
}

// shadowAndFlamePreservesImmolate is Shadow and Flame's other
// Conflagrate half: a 20/40/60/80/100% chance not to consume Immolate.
// Called from conflagrate.go right before it would otherwise deactivate
// the target's Immolate dot.
func (warlock *Warlock) shadowAndFlamePreservesImmolate(sim *core.Simulation) bool {
	rank := warlock.Talents.ShadowAndFlame
	if rank == 0 {
		return false
	}
	return sim.Proc(shadowAndFlameImmolatePreserveChance*float64(rank), "Shadow and Flame")
}

// applyBaneOfHavoc registers the Bane of Havoc curse and the
// damage-redirect hook that follows it: 15% of every OTHER landed hit
// the warlock deals is also dealt to whichever single enemy currently
// carries the curse. Only relevant on a multi-target encounter - the
// single-boss DPS test suites this package ships never exercise the
// redirect itself, only that casting the curse and tracking the aura
// does not panic - but the talent is a real DPS cooldown on an AoE or
// cleave fight, so it is implemented rather than marked unmodelled.
// baneOfHavocManaCostPct is SpellPower.PowerCostPct for spell 1225228 in
// build 1.60.1.70009; see the ManaCost comment in applyBaneOfHavoc.
const baneOfHavocManaCostPct = 5.0

func (warlock *Warlock) applyBaneOfHavoc() {
	if !warlock.Talents.BaneOfHavoc {
		return
	}

	actionID := core.ActionID{SpellID: 1225228}

	warlock.BaneOfHavocAuras = warlock.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    "Bane of Havoc-" + warlock.Label,
			ActionID: actionID,
			Duration: baneOfHavocDuration,
		})
	})

	spillover := warlock.RegisterSpell(core.SpellConfig{
		ActionID:         actionID.WithTag(1),
		SpellSchool:      core.SpellSchoolShadow,
		ProcMask:         core.ProcMaskEmpty,
		Flags:            core.SpellFlagPassiveSpell,
		DamageMultiplier: 1,
		ThreatMultiplier: 0,
	})

	core.MakePermanent(warlock.RegisterAura(core.Aura{
		Label: "Bane of Havoc Trigger",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell == spillover || !result.Landed() || result.Damage <= 0 {
				return
			}

			target := warlock.baneOfHavocTarget
			if target == nil || result.Target == target {
				return
			}
			if !warlock.BaneOfHavocAuras.Get(target).IsActive() {
				return
			}

			// A bare DealDamage on a manually built result, rather than
			// CalcAndDealDamage, because result.Damage here is already
			// the FINAL amount the original hit dealt (every multiplier
			// already applied); routing it back through CalcDamage
			// would apply the warlock's general damage multipliers a
			// second time.
			spilloverResult := spillover.NewResult(target)
			spilloverResult.Outcome = core.OutcomeHit
			spilloverResult.Damage = result.Damage * baneOfHavocSpilloverPct
			spillover.DealDamage(sim, spilloverResult)
		},
	}))

	warlock.BaneOfHavoc = warlock.RegisterSpell(core.SpellConfig{
		ActionID:      actionID,
		SpellSchool:   core.SpellSchoolShadow,
		ProcMask:      core.ProcMaskEmpty,
		Flags:         core.SpellFlagAPL | WarlockFlagDestruction,
		RequiredLevel: 1,

		// 5% of base mana: the client row for THIS id (1225228) carries
		// SpellPower.PowerCostPct 5 (spellconst cost_pct) and a flat cost
		// of 0. The generated BaneOfHavocManaCostPct reads 0 because the
		// generator's rank winner is the sibling id 1243339, which has
		// no power row, so the percentage is taken from 1225228 by hand.
		ManaCost: core.ManaCostOptions{
			BaseCost: baneOfHavocManaCostPct / 100,
		},

		// The client flags Bane of Havoc GCD-less (gcd_ms 0,
		// spellconst/warlock.json build 1.60.1.70009) - it is a
		// cooldown-only curse placement, not a cast - so DefaultCast is
		// left zero-valued instead of taking core.GCDDefault.

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			warlock.baneOfHavocTarget = target
			warlock.BaneOfHavocAuras.Get(target).Activate(sim)
		},
	})
}
