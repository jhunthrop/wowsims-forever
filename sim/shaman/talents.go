package shaman

import (
	"fmt"
	"slices"
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

// Forever's per-rank rates (1.60.1.70291 talents/shaman.json): Ancestral
// Knowledge "Increases your Intellect by 2%" (vanilla: +1% maximum Mana),
// Anticipation "Increases your chance to dodge by an additional 2%"
// (vanilla 1%), Toughness "Increases your Stamina by 2%" (vanilla: armor
// from items) and Flurry "Increases your attack speed by 5%" (vanilla
// 10 + 5 a rank).
const (
	ancestralKnowledgeIntellectPerRank = 0.02
	anticipationDodgePerRank           = 2.0
	toughnessStaminaPerRank            = 0.02
	flurryAttackSpeedPerRank           = 0.05
)

func (shaman *Shaman) ApplyTalents() {
	// Elemental Talents
	shaman.applyConcussion()
	shaman.applyElementalFocus()
	shaman.applyElementalDevastation()
	shaman.applyImprovedFireTotems()
	shaman.applyElementalFury()
	shaman.registerElementalMasteryCD()
	shaman.applyCallOfThunder()
	shaman.applyElementalAlacrity()
	shaman.applyEyeOfTheStorm()

	// Enhancement Talents
	shaman.applyFlurry()
	shaman.applyMentalDexterity()
	shaman.applyMentalQuickness()
	shaman.applyShamanisticFocus()
	shaman.applySpiritWeapons()
	shaman.applyImprovedStormstrike()
	shaman.applyMaelstromWeapon()
	shaman.registerRageOfTheFarseer()

	if shaman.Talents.AncestralKnowledge > 0 {
		shaman.MultiplyStat(stats.Intellect, 1.0+ancestralKnowledgeIntellectPerRank*float64(shaman.Talents.AncestralKnowledge))
	}

	/*
		shaman.AddStat(stats.Block, 1*float64(shaman.Talents.ShieldSpecialization))
	*/

	shaman.AddStat(stats.Crit, core.CritRatingPerCritChance*1*float64(shaman.Talents.ThunderingStrikes))

	shaman.AddStat(stats.Dodge, anticipationDodgePerRank*float64(shaman.Talents.Anticipation))

	shaman.MultiplyStat(stats.Stamina, 1+toughnessStaminaPerRank*float64(shaman.Talents.Toughness))

	/*
		if shaman.Talents.Parry {
			shaman.PseudoStats.CanParry = true
		}
	*/

	// TODO: Check whether this does what it should.
	// From all I've seen this appears to not actually be a school modifier at all, but instead simply applies
	// to all attacks done with a weapon. The weaponmask seems to take precedence and the school mask is actually ignored.
	// Will also be the case for similar talents like the one for retribution.
	/*
		shaman.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= 1 + (.02 * float64(shaman.Talents.WeaponMastery))
	*/

	// Restoration Talents
	shaman.applyRestorationTalents()
	shaman.registerNaturesSwiftnessCD()
	shaman.registerWaterShieldSpell()

	shaman.applyUnmodeledTalents()
}

func (shaman *Shaman) applyConcussion() {
	if shaman.Talents.Concussion == 0 {
		return
	}

	additiveMultiplier := 0.01 * float64(shaman.Talents.Concussion)
	// The client's Concussion text and class mask (spell 16035) name
	// Lightning Bolt, Chain Lightning and Earth Shock only; Flame Shock
	// and Frost Shock sat in this list from the Classic tree and drew a
	// bonus they do not have.
	affectedSpellCodes := []int32{SpellCode_ShamanLightningBolt, SpellCode_ShamanChainLightning, SpellCode_ShamanEarthShock}

	shaman.OnSpellRegistered(func(spell *core.Spell) {
		if slices.Contains(affectedSpellCodes, spell.SpellCode) {
			spell.DamageMultiplierAdditive += additiveMultiplier
		}
	})
}
func (shaman *Shaman) callOfFlameMultiplier() float64 {
	return 1 + .05*float64(shaman.Talents.CallOfFlame)
}

func (shaman *Shaman) applyElementalFocus() {
	if !shaman.Talents.ElementalFocus {
		return
	}

	var affectedSpells []*core.Spell

	shaman.ClearcastingAura = shaman.RegisterAura(core.Aura{
		Label:     "Clearcasting",
		ActionID:  core.ActionID{SpellID: 16246},
		Duration:  time.Second * 15,
		MaxStacks: 1,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			affectedSpells = shaman.getClearcastingSpells()
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			core.Each(affectedSpells, func(spell *core.Spell) {
				if spell.Cost != nil {
					spell.Cost.Multiplier -= 100
				}
			})
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			core.Each(affectedSpells, func(spell *core.Spell) {
				if spell.Cost != nil {
					spell.Cost.Multiplier += 100
				}
			})
		},
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks, newStacks int32) {
			if newStacks == 0 {
				aura.Deactivate(sim)
			}
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			// OnCastComplete is called after OnSpellHitDealt / etc, so don't deactivate if it was just activated.
			if aura.RemainingDuration(sim) == aura.Duration {
				return
			}

			if aura.GetStacks() > 0 && shaman.isShamanDamagingSpell(spell) {
				aura.RemoveStack(sim)
			}
		},
	})

	core.MakePermanent(shaman.RegisterAura(core.Aura{
		Label: "Elemental Focus Trigger",
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if shaman.isShamanDamagingSpell(spell) && sim.Proc(0.10, "Elemental Focus") {
				shaman.ClearcastingAura.Activate(sim)
				shaman.ClearcastingAura.SetStacks(sim, shaman.ClearcastingAura.MaxStacks)
			}
		},
	}))
}

func (shaman *Shaman) isShamanDamagingSpell(spell *core.Spell) bool {
	return spell.Flags.Matches(SpellFlagShaman) && spell.ProcMask.Matches(core.ProcMaskSpellDamage)
}

func (shaman *Shaman) getClearcastingSpells() []*core.Spell {
	return core.FilterSlice(
		shaman.Spellbook,
		func(spell *core.Spell) bool {
			return spell != nil && shaman.isShamanDamagingSpell(spell)
		},
	)
}

func (shaman *Shaman) applyElementalDevastation() {
	if shaman.Talents.ElementalDevastation == 0 {
		return
	}

	spellID := []int32{0, 30165, 29177, 29178}[rankIndex(shaman.Talents.ElementalDevastation, 4)]
	critBonus := 3.0 * float64(shaman.Talents.ElementalDevastation) * core.CritRatingPerCritChance
	// The client's text and effect (spell 30165, apply aura 52, melee crit)
	// grant MELEE crit chance only. Forever's Crit stat is unified across
	// spell, melee and ranged, so the stat bonus alone would also raise
	// the shaman's own spell crit - the bonus an Elemental build was being
	// credited for - and the proc takes the same amount back off every
	// spell school while it is up.
	procAura := shaman.NewTemporaryStatsAuraWrapped("Elemental Devastation Proc", core.ActionID{SpellID: spellID}, stats.Stats{stats.Crit: critBonus}, time.Second*10, func(aura *core.Aura) {
		gainStat, loseStat := aura.OnGain, aura.OnExpire
		aura.OnGain = func(aura *core.Aura, sim *core.Simulation) {
			gainStat(aura, sim)
			shaman.addSpellSchoolCrit(-critBonus)
		}
		aura.OnExpire = func(aura *core.Aura, sim *core.Simulation) {
			loseStat(aura, sim)
			shaman.addSpellSchoolCrit(critBonus)
		}
	})

	shaman.RegisterAura(core.Aura{
		Label:    "Elemental Devastation",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.ProcMask.Matches(core.ProcMaskSpellDamage) && result.Outcome.Matches(core.OutcomeCrit) {
				procAura.Activate(sim)
			}
		},
	})
}

// addSpellSchoolCrit adds critRating to every magic school's crit bonus
// (the per-school term SpellCritChance reads on top of the unified Crit
// stat), leaving melee and ranged crit alone.
func (shaman *Shaman) addSpellSchoolCrit(critRating float64) {
	shaman.PseudoStats.SchoolBonusCritChance.AddToMagicSchools(critRating)
}

func (shaman *Shaman) applyImprovedFireTotems() {
	/*
		if shaman.Talents.ImprovedFireTotems == 0 {
			return
		}

		shaman.OnSpellRegistered(func(spell *core.Spell) {
			if spell.SpellCode == SpellCode_ShamanFireNova {
				for _, dot := range spell.Dots() {
					if dot == nil {
						continue
					}

					dot.TickLength -= time.Second * time.Duration(shaman.Talents.ImprovedFireTotems)
				}
			} else if spell.SpellCode == SpellCode_ShamanMagmaTotem {
				spell.ThreatMultiplier *= 1.0 - (0.25 * float64(shaman.Talents.ImprovedFireTotems))
			}
		})
	*/
}

const (
	elementalFuryCritDamageBonusPerRank = 0.2
	elementalFuryMaxRank                = 5
)

func clampRank(rank int32, maxRank int32) int32 {
	return min(max(rank, 0), maxRank)
}

func (shaman *Shaman) applyElementalFury() {
	// FOREVER: Elemental Fury has ranks in the client's trees, so the field
	// is an int32 now rather than a bool.
	if shaman.Talents.ElementalFury == 0 {
		return
	}

	// "Increases the critical strike damage bonus ... by 20%" per rank, five
	// ranks, so +100% only at 5/5; the Classic one-rank +100% it replaced
	// gave every rank the full bonus.
	critDamageBonus := elementalFuryCritDamageBonusPerRank * float64(clampRank(shaman.Talents.ElementalFury, elementalFuryMaxRank))

	shaman.OnSpellRegistered(func(spell *core.Spell) {
		if elementalFuryApplies(spell) {
			spell.CritDamageBonus += critDamageBonus
		}
	})
}

// elementalFuryApplies reports whether Elemental Fury (16089) raises the
// spell's crit damage bonus. The client's class mask names the shaman's
// Fire, Frost and Nature damage spells and the Searing Totem's and Magma
// Totem's damage, which is dealt by child spells (the Searing Totem's
// Attack, the Magma Totem's pulse) that carry no shaman or totem flag;
// it does not name the healing spells.
func elementalFuryApplies(spell *core.Spell) bool {
	if spell.DefenseType != core.DefenseTypeMagic || spell.ProcMask.Matches(core.ProcMaskSpellHealing) {
		return false
	}
	switch spell.SpellCode {
	case SpellCode_ShamanSearingTotemAttack, SpellCode_ShamanMagmaTotem:
		return true
	}
	return spell.Flags.Matches(SpellFlagShaman)
}

func (shaman *Shaman) registerElementalMasteryCD() {
	/*
		if !shaman.Talents.ElementalMastery {
			return
		}

		actionID := core.ActionID{SpellID: 16166}

		cdTimer := shaman.NewTimer()
		cd := time.Minute * 3

		var affectedSpells []*core.Spell

		emAura := shaman.RegisterAura(core.Aura{
			Label:    "Elemental Mastery",
			ActionID: actionID,
			Duration: core.NeverExpires,
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				affectedSpells = core.FilterSlice(
					shaman.Spellbook,
					func(spell *core.Spell) bool { return spell != nil && shaman.isShamanDamagingSpell(spell) },
				)
			},
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				core.Each(affectedSpells, func(spell *core.Spell) {
					spell.BonusCritRating += core.CritRatingPerCritChance * 100
					if spell.Cost != nil {
						spell.Cost.Multiplier -= 100
					}
				})
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				core.Each(affectedSpells, func(spell *core.Spell) {
					spell.BonusCritRating -= core.CritRatingPerCritChance * 100
					if spell.Cost != nil {
						spell.Cost.Multiplier += 100
					}
				})
				shaman.ElementalMastery.CD.Use(sim)
			},
			OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
				if shaman.isShamanDamagingSpell(spell) {
					// Elemental mastery can be batched
					core.StartDelayedAction(sim, core.DelayedActionOptions{
						DoAt: sim.CurrentTime + core.SpellBatchWindow,
						OnAction: func(sim *core.Simulation) {
							if aura.IsActive() {
								// Remove the buff and put skill on CD
								aura.Deactivate(sim)
								cdTimer.Set(sim.CurrentTime + cd)
								shaman.UpdateMajorCooldowns()
							}
						},
					})
				}
			},
		})

		shaman.ElementalMastery = shaman.RegisterSpell(core.SpellConfig{
			ActionID: actionID,
			Flags:    core.SpellFlagNoOnCastComplete,
			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    cdTimer,
					Duration: cd,
				},
			},
			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
				emAura.Activate(sim)
			},
		})

		shaman.AddMajorCooldown(core.MajorCooldown{
			Spell: shaman.ElementalMastery,
			Type:  core.CooldownTypeDPS,
		})
	*/
}

func (shaman *Shaman) registerNaturesSwiftnessCD() {
	if !shaman.Talents.NaturesSwiftness {
		return
	}
	actionID := core.ActionID{SpellID: 16188}
	cdTimer := shaman.NewTimer()
	cd := time.Minute * 3

	var affectedSpells []*core.Spell

	nsAura := shaman.RegisterAura(core.Aura{
		Label:    "Natures Swiftness",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			affectedSpells = core.FilterSlice(
				shaman.Spellbook,
				func(spell *core.Spell) bool {
					return spell != nil && spell.SpellSchool.Matches(core.SpellSchoolNature) && spell.DefaultCast.CastTime > 0
				},
			)
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			core.Each(affectedSpells, func(spell *core.Spell) { spell.CastTimeMultiplier -= 1 })
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			core.Each(affectedSpells, func(spell *core.Spell) { spell.CastTimeMultiplier += 1 })
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.SpellSchool.Matches(core.SpellSchoolNature) && spell.DefaultCast.CastTime > 0 {
				// Remove the buff and put skill on CD
				aura.Deactivate(sim)
				cdTimer.Set(sim.CurrentTime + cd)
				shaman.UpdateMajorCooldowns()
			}
		},
	})

	nsSpell := shaman.RegisterSpell(core.SpellConfig{
		ActionID:      actionID,
		Flags:         core.SpellFlagNoOnCastComplete,
		RequiredLevel: 1,
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: cd,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			// A temporary cast speed buff already shortens the cast NS would
			// make instant, so the cooldown is better kept.
			return !shaman.HasTemporarySpellCastSpeedIncrease()
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			nsAura.Activate(sim)
		},
	})

	shaman.AddMajorCooldown(core.MajorCooldown{
		Spell: nsSpell,
		Type:  core.CooldownTypeDPS,
		// A healer spends Nature's Swiftness by hand on an emergency heal;
		// autocast would burn it on the first cast of the fight.
		ShouldActivate: func(_ *core.Simulation, _ *core.Character) bool {
			return len(shaman.HealingWave) == 0
		},
	})
}

func (shaman *Shaman) applyFlurry() {
	if shaman.Talents.Flurry == 0 {
		return
	}

	talentAura := shaman.makeFlurryAura(shaman.Talents.Flurry)

	// This must be registered before the below trigger because in-game a crit weapon swing consumes a stack before the refresh, so you end up with:
	// 3 => 2
	// refresh
	// 2 => 3
	shaman.makeFlurryConsumptionTrigger(talentAura)

	shaman.RegisterAura(core.Aura{
		Label:    "Flurry Proc Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.ProcMask.Matches(core.ProcMaskMelee) && result.Outcome.Matches(core.OutcomeCrit) {
				talentAura.Activate(sim)
				if talentAura.IsActive() {
					talentAura.SetStacks(sim, 3)
				}
				return
			}
		},
	})
}

// These are separated out because of the T1 Shaman Tank 2P that can proc Flurry separately from the talent.
// It triggers the max-rank Flurry aura but with dodge, parry, or block.
// flurryAttackSpeed is the attack speed multiplier Flurry grants at a rank.
func flurryAttackSpeed(points int32) float64 {
	return 1 + flurryAttackSpeedPerRank*float64(points)
}

func (shaman *Shaman) makeFlurryAura(points int32) *core.Aura {
	if points == 0 {
		return nil
	}

	spellID := []int32{16257, 16277, 16278, 16279, 16280}[points-1]
	attackSpeed := flurryAttackSpeed(points)

	aura := shaman.GetOrRegisterAura(core.Aura{
		Label:     fmt.Sprintf("Flurry Proc (%d)", spellID),
		ActionID:  core.ActionID{SpellID: spellID},
		Duration:  core.NeverExpires,
		MaxStacks: 3,
	})

	aura.NewExclusiveEffect("Flurry", true, core.ExclusiveEffect{
		Priority: attackSpeed,
		OnGain: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			shaman.MultiplyMeleeSpeed(sim, attackSpeed)
		},
		OnExpire: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			shaman.MultiplyMeleeSpeed(sim, 1/(attackSpeed))
		},
	})

	return aura
}

// With the Warden T1 2pc it's possible to have 2 different Flurry auras if using less than 5/5 points in Flurry.
// The two different buffs don't stack whatsoever. Instead the stronger aura takes precedence and each one is only refreshed by the corresponding triggers.
func (shaman *Shaman) makeFlurryConsumptionTrigger(flurryAura *core.Aura) *core.Aura {
	icd := core.Cooldown{
		Timer:    shaman.NewTimer(),
		Duration: time.Millisecond * 500,
	}
	return core.MakePermanent(shaman.GetOrRegisterAura(core.Aura{
		Label: fmt.Sprintf("Flurry Consume Trigger - %d", flurryAura.ActionID.SpellID),
		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// Remove a stack.
			if flurryAura.IsActive() && spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) && icd.IsReady(sim) {
				icd.Use(sim)
				flurryAura.RemoveStack(sim)
			}
		},
	}))
}

func (shaman *Shaman) totemManaMultiplier() int32 {
	return 100 - 5*shaman.Talents.TotemicFocus
}

// callOfThunderCritBonus is Call of Thunder's one rank (talents/shaman.json
// node 104762, max_rank 1): "Increases the critical strike chance of
// your Lightning Bolt and Chain Lightning spells by 3%." Lightning
// Shield's own cast and proc spells also carry SpellFlagLightning, so
// this matches on SpellCode rather than that flag, the same way
// applyConcussion already has to for the shared affected-spell list.
const callOfThunderCritBonus = 3 * core.CritRatingPerCritChance

func (shaman *Shaman) applyCallOfThunder() {
	if !shaman.Talents.CallOfThunder {
		return
	}

	affectedSpellCodes := []int32{SpellCode_ShamanLightningBolt, SpellCode_ShamanChainLightning}
	shaman.OnSpellRegistered(func(spell *core.Spell) {
		if slices.Contains(affectedSpellCodes, spell.SpellCode) {
			spell.BonusCritRating += callOfThunderCritBonus
		}
	})
}

// elementalAlacrityCastTimeReduction is Elemental Alacrity's three
// ranks (talents/shaman.json node 104765): "Reduces the cast time of
// your Lightning Bolt, Chain Lightning, and Lava Burst spells by 0.17/
// 0.33/0.5 sec." A flat reduction to the pre-haste base cast time,
// applied once at registration - not a percentage, so it cannot reuse
// CastTimeMultiplier the way Elemental Mastery and Nature's Swiftness
// do above.
var elementalAlacrityCastTimeReduction = [4]time.Duration{0, 170 * time.Millisecond, 330 * time.Millisecond, 500 * time.Millisecond}

func (shaman *Shaman) applyElementalAlacrity() {
	if shaman.Talents.ElementalAlacrity == 0 {
		return
	}

	reduction := elementalAlacrityCastTimeReduction[rankIndex(shaman.Talents.ElementalAlacrity, len(elementalAlacrityCastTimeReduction))]
	affectedSpellCodes := []int32{SpellCode_ShamanLightningBolt, SpellCode_ShamanChainLightning, SpellCode_ShamanLavaBurst}
	shaman.OnSpellRegistered(func(spell *core.Spell) {
		if slices.Contains(affectedSpellCodes, spell.SpellCode) && spell.DefaultCast.CastTime > 0 {
			spell.DefaultCast.CastTime = max(0, spell.DefaultCast.CastTime-reduction)
		}
	})
}

// eyeOfTheStormPushbackReduction is Eye of the Storm's three ranks
// (talents/shaman.json node 104763): "Reduces the pushback suffered
// from damaging attacks while casting Lightning Bolt, Chain Lightning,
// and Lava Burst by 23/47/70%." sim/core/cast.go's applySpellPushback
// already reads Spell.PushbackReduction as exactly this chance to
// avoid a pushback entirely; this talent is the first to ever set it.
var eyeOfTheStormPushbackReduction = [4]float64{0, 0.23, 0.47, 0.70}

func (shaman *Shaman) applyEyeOfTheStorm() {
	if shaman.Talents.EyeOfTheStorm == 0 {
		return
	}

	reduction := eyeOfTheStormPushbackReduction[rankIndex(shaman.Talents.EyeOfTheStorm, len(eyeOfTheStormPushbackReduction))]
	affectedSpellCodes := []int32{SpellCode_ShamanLightningBolt, SpellCode_ShamanChainLightning, SpellCode_ShamanLavaBurst}
	shaman.OnSpellRegistered(func(spell *core.Spell) {
		if slices.Contains(affectedSpellCodes, spell.SpellCode) {
			spell.PushbackReduction += reduction
		}
	})
}

// lightningOverloadChance is Lightning Overload's three ranks
// (talents/shaman.json node 104759): "Gives your Lightning Bolt and
// Chain Lightning spells a 3/7/10% chance to cast a second, similar
// spell on the same target at no additional cost that causes half
// damage and no threat." lightning_bolt.go and chain_lightning.go call
// rollLightningOverload from inside their own ApplyEffects and, on a
// proc, deal a second hit through the SAME spell object at half the
// rolled base damage, rather than this file registering a cloned
// spell: a second call through the original spell automatically
// carries every multiplier ("a second, similar spell") the primary
// hit already has - Concussion, Call of Thunder, Elemental Fury, gear,
// buffs - instead of a clone that has to be kept in sync by hand. The
// tooltip's "no threat" has no DPS effect in this engine (threat is
// tracked separately from the damage the sim reports) and is not
// modeled.
var lightningOverloadChance = [4]float64{0, 0.03, 0.07, 0.10}

// lightningOverloadDamageScale is the whole second hit's share of the
// first: the client's overload spells (408477 for Lightning Bolt rank 10,
// 408484 for Chain Lightning rank 4) carry half the primary's amount AND
// half its spell-power coefficient (98 at 0.357 against 196 at 0.714), so
// halving only the base damage, as the first version did, let the
// overload keep the full spell-power share.
const lightningOverloadDamageScale = 0.5

// AtLightningOverloadScale runs hit with spell's damage multiplier cut to
// the overload's share, then restores it. Exported so the elemental
// package's tests can check the scale without forcing the proc.
func (shaman *Shaman) AtLightningOverloadScale(spell *core.Spell, hit func()) {
	enteringMultiplier := spell.DamageMultiplier
	spell.DamageMultiplier *= lightningOverloadDamageScale
	hit()
	spell.DamageMultiplier = enteringMultiplier
}

func (shaman *Shaman) rollLightningOverload(sim *core.Simulation) bool {
	if shaman.Talents.LightningOverload == 0 {
		return false
	}
	return sim.Proc(lightningOverloadChance[rankIndex(shaman.Talents.LightningOverload, len(lightningOverloadChance))], "Lightning Overload")
}

// mentalDexterityAttackPowerPercent is Mental Dexterity's three ranks
// (talents/shaman.json node 104755): "Increases your Attack Power by
// an amount equal to 33/67/100% of your Intellect."
var mentalDexterityAttackPowerPercent = [4]float64{0, 0.33, 0.67, 1.0}

func (shaman *Shaman) applyMentalDexterity() {
	if shaman.Talents.MentalDexterity == 0 {
		return
	}
	shaman.AddStatDependency(stats.Intellect, stats.AttackPower, mentalDexterityAttackPowerPercent[rankIndex(shaman.Talents.MentalDexterity, len(mentalDexterityAttackPowerPercent))])
}

// mentalQuicknessSpellDamagePercent is Mental Quickness's two ranks
// (talents/shaman.json node 104744): "Increases your spell damage and
// healing by up to 15/30% of your Intellect." Both stats.SpellDamage
// and stats.HealingPower get the dependency because the tooltip names
// both; HealingPower already carries a further third of itself into
// SpellDamage (sim/core/character.go's HealingToSpellDamageRatio
// dependency, applied to every character), the same as gear's own
// healing power already does.
var mentalQuicknessSpellDamagePercent = [3]float64{0, 0.15, 0.30}

func (shaman *Shaman) applyMentalQuickness() {
	if shaman.Talents.MentalQuickness == 0 {
		return
	}
	pct := mentalQuicknessSpellDamagePercent[rankIndex(shaman.Talents.MentalQuickness, len(mentalQuicknessSpellDamagePercent))]
	shaman.AddStatDependency(stats.Intellect, stats.SpellDamage, pct)
	shaman.AddStatDependency(stats.Intellect, stats.HealingPower, pct)
}

// applyShamanisticFocus is Shamanistic Focus's one rank
// (talents/shaman.json node 104749, max_rank 1): "Reduces the mana
// cost of your Shock and Lightning Shield spells by 45%." Matches the
// same subtractive-percentage-points-from-100 convention Convection's
// own ManaCostOptions.Multiplier already uses in shocks.go, applied
// after the fact so it stacks with Convection instead of overwriting it.
func (shaman *Shaman) applyShamanisticFocus() {
	if !shaman.Talents.ShamanisticFocus {
		return
	}

	affectedSpellCodes := []int32{SpellCode_ShamanEarthShock, SpellCode_ShamanFlameShock, SpellCode_ShamanFrostShock, SpellCode_ShamanLightningShield}
	shaman.OnSpellRegistered(func(spell *core.Spell) {
		if slices.Contains(affectedSpellCodes, spell.SpellCode) && spell.Cost != nil {
			spell.Cost.Multiplier -= 45
		}
	})
}

// applySpiritWeapons is Spirit Weapons's one rank (talents/shaman.json
// node 104745, max_rank 1): "Gives a chance to parry enemy melee
// attacks, reduces all threat generated by your attacks by 30% while
// Rockbiter Weapon is not active, and increases all threat generated
// by 30% while Rockbiter Weapon is active." Only the parry grant is
// modeled: sim/core/character.go already gives every character a
// baseline 5% stats.Parry that only matters once PseudoStats.CanParry
// is true, and this is Elemental/Enhancement DPS-relevant because
// Improved Stormstrike's cooldown reset triggers on exactly this -
// the shaman's own Dodge or Parry outcome against an incoming attack
// (applyImprovedStormstrike below). The threat halves are a threat
// mechanic this engine's DPS sim does not compute and are not modeled.
func (shaman *Shaman) applySpiritWeapons() {
	if !shaman.Talents.SpiritWeapons {
		return
	}
	shaman.PseudoStats.CanParry = true
}

// improvedStormstrikeChance is Improved Stormstrike's two ranks
// (talents/shaman.json node 104742, prereq Stormstrike 1/1): "When you
// Stormstrike, you have a 50/100% chance to gain 50% mana regeneration
// while casting spells for 15 sec, and Stormstrike's cooldown has a
// 50/100% chance to reset each time you Dodge or Parry." Both halves
// share one roll chance per rank in the client text, so one table
// drives both triggers below.
var improvedStormstrikeChance = [3]float64{0, 0.5, 1.0}

func (shaman *Shaman) applyImprovedStormstrike() {
	if shaman.Talents.ImprovedStormstrike == 0 {
		return
	}
	chance := improvedStormstrikeChance[rankIndex(shaman.Talents.ImprovedStormstrike, len(improvedStormstrikeChance))]

	// "50% mana regeneration while casting" is the same mechanic
	// Priest's Meditation/Shadow Form grant through
	// PseudoStats.SpiritRegenRateCasting (sim/priest/talents.go); here
	// it is a 15-second buff rather than a passive, triggered off
	// Stormstrike rather than always active.
	manaRegenAura := shaman.RegisterAura(core.Aura{
		Label:    "Improved Stormstrike",
		ActionID: core.ActionID{SpellID: ImprovedStormstrikeSpellId[0]},
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			shaman.PseudoStats.SpiritRegenRateCasting += 0.5
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			shaman.PseudoStats.SpiritRegenRateCasting -= 0.5
		},
	})

	core.MakePermanent(shaman.RegisterAura(core.Aura{
		Label: "Improved Stormstrike Trigger",
		OnCastComplete: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.SpellCode == SpellCode_ShamanStormstrike && sim.Proc(chance, "Improved Stormstrike Mana Regen") {
				manaRegenAura.Activate(sim)
			}
		},
		OnSpellHitTaken: func(_ *core.Aura, sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			if shaman.Stormstrike == nil || !result.Outcome.Matches(core.OutcomeDodge|core.OutcomeParry) {
				return
			}
			if sim.Proc(chance, "Improved Stormstrike Reset") {
				shaman.Stormstrike.CD.Timer.Set(sim.CurrentTime)
			}
		},
	}))
}

// maelstromWeaponProcChance is datamined, not from the rank text:
// spellconst/shaman.json's "Lightning Bolt Can Activate Maelstrom
// Weapon" (spell 1291078) carries amount 50, the same 50 that spell
// 408498's own second effect carries, and the rank text itself never
// states a chance ("you have a chance to reduce..."). 408498 is used
// as the per-rank display id (matching talents_auto_gen.go's
// TalentSpellIDs, not the "Maelstrom Weapon"-named spell 409946 the
// constants generator picked instead - 409946 is a wrapper that just
// applies 408498, per its own aura/amount pair in that same file).
//
// maelstromWeaponPerStackPercent is the five ranks (talents/shaman.json
// node 104741): "...reduce the cast time and Mana cost of your next
// Lightning Bolt spell by 4/8/12/16/20%. Stacks up to 5 times." Read
// literally, each stack contributes the rank's own percentage, so
// rank 5 at 5 stacks is a 100% reduction - an instant, free Lightning
// Bolt - consistent with "Stacks up to 5 times" compounding rather
// than capping at the rank's own number.
const maelstromWeaponProcChance = 0.5
const maelstromWeaponDisplaySpellId = 408498
const maelstromWeaponMaxStacks = int32(5)

// maelstromWeaponChance is the chance a damaging hit by spell grants a
// Maelstrom Weapon stack: a melee hit's own, or Lightning Bolt's reduced one
// once Totem of the Storm lets it trigger the talent.
func (shaman *Shaman) maelstromWeaponChance(spell *core.Spell) float64 {
	switch {
	case spell.ProcMask.Matches(core.ProcMaskMelee):
		return maelstromWeaponProcChance
	case spell.SpellCode == SpellCode_ShamanLightningBolt:
		return shaman.LightningBoltMaelstromChance
	default:
		return 0
	}
}

func (shaman *Shaman) applyMaelstromWeapon() {
	if shaman.Talents.MaelstromWeapon == 0 {
		return
	}

	perStackPercent := int32(4 * shaman.Talents.MaelstromWeapon)
	perStackCastFraction := float64(perStackPercent) / 100.0

	var lightningBoltSpells []*core.Spell
	shaman.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_ShamanLightningBolt {
			lightningBoltSpells = append(lightningBoltSpells, spell)
		}
	})

	maelstromAura := shaman.RegisterAura(core.Aura{
		Label:     "Maelstrom Weapon",
		ActionID:  core.ActionID{SpellID: maelstromWeaponDisplaySpellId},
		Duration:  time.Second * 30,
		MaxStacks: maelstromWeaponMaxStacks,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks, newStacks int32) {
			delta := newStacks - oldStacks
			for _, spell := range lightningBoltSpells {
				if spell.Cost != nil {
					spell.Cost.Multiplier -= delta * perStackPercent
				}
				spell.CastTimeMultiplier -= float64(delta) * perStackCastFraction
			}
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			// Mirrors ClearcastingAura's own guard (applyElementalFocus
			// above): OnCastComplete runs for every active aura after
			// the triggering cast too, so don't consume the buff on the
			// same event that granted it.
			if aura.RemainingDuration(sim) == aura.Duration {
				return
			}
			if spell.SpellCode != SpellCode_ShamanLightningBolt {
				return
			}
			aura.Deactivate(sim)
		},
	})
	shaman.MaelstromWeaponAura = maelstromAura

	core.MakePermanent(shaman.RegisterAura(core.Aura{
		Label: "Maelstrom Weapon Trigger",
		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			chance := shaman.maelstromWeaponChance(spell)
			if chance == 0 || result.Damage <= 0 {
				return
			}
			if !sim.Proc(chance, "Maelstrom Weapon") {
				return
			}
			maelstromAura.Activate(sim)
			maelstromAura.AddStack(sim)
		},
	}))
}

// registerRageOfTheFarseer is the Enhancement tree's tier-6 bool
// talent (talents/shaman.json node 104740, max_rank 1, prereq Mental
// Quickness 2/2): "Increases your attack speed by 30% for 25 sec." -
// an activated burst cooldown with a 180000ms (3 min) cooldown that is
// datamined (spellconst/shaman.json spell 425336's SpellCooldowns.csv
// row) rather than stated in the rank text. Melee attack speed only:
// an earlier revision also raised casting speed, on the strength of
// pre-beta datamining, and Forever's 24 September 2026 beta notes
// removed exactly that ("Rage of the Farseer no longer speeds up your
// spellcasting"); build 1.60.1.70009's own rank text names attack
// speed alone.
func (shaman *Shaman) registerRageOfTheFarseer() {
	if !shaman.Talents.RageOfTheFarseer {
		return
	}

	actionID := core.ActionID{SpellID: RageOfTheFarseerSpellId[0]}
	const hasteBonus = 1.30
	cdTimer := shaman.NewTimer()

	rotfAura := shaman.RegisterAura(core.Aura{
		Label:    "Rage of the Farseer",
		ActionID: actionID,
		Duration: time.Second * 25,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			shaman.MultiplyAttackSpeed(sim, hasteBonus)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			shaman.MultiplyAttackSpeed(sim, 1/hasteBonus)
		},
	})

	shaman.RageOfTheFarseer = shaman.RegisterSpell(core.SpellConfig{
		ActionID:        actionID,
		Flags:           SpellFlagShaman | core.SpellFlagAPL | core.SpellFlagNoOnCastComplete,
		RequiredLevel:   1,
		RelatedSelfBuff: rotfAura,
		Cast: core.CastConfig{
			// FOREVER: spellconst/shaman.json spell 425336's gcd_ms is 0 -
			// a cooldown-only burst ability the client flags GCD-less, the
			// same shape as Warlock's Bane of Havoc.
			DefaultCast: core.Cast{GCD: 0},
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: time.Minute * 3,
			},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			rotfAura.Activate(sim)
		},
	})

	shaman.AddMajorCooldown(core.MajorCooldown{
		Spell: shaman.RageOfTheFarseer,
		Type:  core.CooldownTypeDPS,
	})
}

// applyUnmodeledTalents reads every remaining talent proto field this
// package does not otherwise touch, each with the reason it is not
// worth a DPS-sim behaviour. "Every talent the engine does not model
// is named with the reason rather than dropped" (AGENTS.md).
func (shaman *Shaman) applyUnmodeledTalents() {
	t := shaman.Talents
	_ = t.ElementalWarding      // reduces damage taken from Fire/Frost/Nature; no outgoing-DPS effect.
	_ = t.ElementalReach        // spell range and Flame Shock range only; the sim has no positioning.
	_ = t.Earthbound            // Earthbind Totem immobilize is a CC utility, not damage.
	_ = t.EarthsGrasp           // Stoneclaw Totem health and Earthbind Totem radius; no DPS effect.
	_ = t.ImprovedGhostWolf     // Ghost Wolf cast time and indoor use; a travel buff, not combat.
	_ = t.ImprovedReincarnation // Reincarnation cooldown and return amount; the sim has no death, so no ankh.
	_ = t.AncestralHealing      // +armor on a healed target for 15 s after a healing crit; the raid damage model's hits are plain health loss that ignores armor, so it changes no healing outcome.
	_ = t.HealingFocus          // avoids damage pushback while casting healing spells; the engine's pushback rolls the attacker's spell, and no fake raid damage ever hits the healer.
}

// rankIndex clamps a talent rank to a per-rank table's last entry, the
// same guard sim/mage/talents.go carries: a talent string written for an
// older tree can carry a rank the current table does not have, and a
// panic there takes the whole character build down with it.
func rankIndex(rank int32, tableLen int) int {
	if i := int(rank); i >= 0 && i < tableLen {
		return i
	}
	return tableLen - 1
}
