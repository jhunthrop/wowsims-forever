package priest

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

func (priest *Priest) ApplyTalents() {
	// Discipline
	priest.registerInnerFocus()
	priest.applyDivineAegis()

	priest.applySilentResolve()

	/*
		if priest.Talents.ImprovedPowerWordFortitude > 0 {
			priest.MultiplyStat(stats.Stamina, 1.0+.15*float64(priest.Talents.ImprovedPowerWordFortitude))
		}
	*/

	priest.PseudoStats.SpiritRegenRateCasting = []float64{0.0, 0.17, 0.33, 0.5}[priest.Talents.Meditation]

	if priest.Talents.MentalStrength > 0 {
		priest.MultiplyStat(stats.Intellect, 1.0+mentalStrengthIntellectPerRank*float64(priest.Talents.MentalStrength))
	}

	// Power in Light: "Your Smite and Penance spells deal X% increased
	// damage to targets afflicted with your Holy Fire." Damage only, and
	// the healing specs cast Penance on allies; a healer's numbers do not
	// move. Named, not modeled.
	_ = priest.Talents.PowerInLight

	// Holy Precision: "Improves your chance to hit with Holy spells by
	// X%." Heals do not roll to hit; only Smite and Holy Fire do, and no
	// healing rotation casts them. Named, not modeled.
	_ = priest.Talents.HolyPrecision

	// Martyrdom: a chance, on being critically struck, to resist
	// pushback and interrupt effects for 6 sec. The only lever core has
	// for pushback resistance (Spell.PushbackReduction) is read from the
	// ATTACKING spell in applySpellPushback (sim/core/cast.go), not from
	// anything a target-side talent can set, so there is no mod kind
	// this talent could use without a core change. Named, not modeled.
	_ = priest.Talents.Martyrdom

	// Improved Mana Burn: shortens Mana Burn's cast time. Mana Burn is
	// not registered in this package (a PvP ability with no mana-pool
	// target on a Patchwerk-style dummy). Named, not modeled.
	_ = priest.Talents.ImprovedManaBurn

	// Twilight Focus: "a X% chance to avoid interruption caused by
	// damage while casting any spell." Same pushback gap as Martyrdom:
	// core reads the lever off the attacking spell, and the fake raid
	// never interrupts the healer anyway. Named, not modeled; reported as
	// a core gap.
	_ = priest.Talents.TwilightFocus

	// Holy
	priest.applyInspiration()
	priest.applyHolySpecialization()
	priest.applySearingLight()
	priest.applyLitanyOfLight()

	priest.PseudoStats.SchoolDamageTakenMultiplier.MultiplyMagicSchools(1 - 0.02*float64(priest.Talents.SpellWarding))

	if priest.Talents.SpiritualGuidance > 0 {
		// "Increases your spell healing by up to 5% of your total Spirit
		// and your spell damage by up to 1% of your total Spirit" per
		// rank. Spirit grants the healing stat; the damage half is
		// Forever's one-third rule (core.HealingToSpellDamageRatio), which
		// already turns the healing stat into spell damage.
		priest.AddStatDependency(stats.Spirit, stats.HealingPower, spiritualGuidanceHealingPerSpirit*float64(priest.Talents.SpiritualGuidance))
	}

	// Blessed Recovery: a heal over time on the priest after a critical
	// hit or a hit of over 30% of its health. The fake raid never hits
	// the healer, so there is no trigger. Named, not modeled.
	_ = priest.Talents.BlessedRecovery

	// Holy Reach: Smite and Holy Fire range and Prayer of Healing and
	// Holy Nova radius. The fake raid stands in range of everything, so
	// nothing changes. Named, not modeled.
	_ = priest.Talents.HolyReach

	// Spirit of Redemption: a 15 s healing form on the priest's death. A
	// healing sim does not end the healer; the priest is not damaged by
	// the raid damage model. Named, not modeled.
	_ = priest.Talents.SpiritOfRedemption

	priest.applyDeclarativeHealingTalents()

	// Shadow
	priest.registerVampiricEmbraceSpell()
	priest.registerShadowform()
	priest.applySpiritTap()
	priest.applyShadowAffinity()
	priest.applyShadowFocus()
	priest.applyShadowWeaving()
	priest.applyDarkness()
	priest.applyDeclarativeShadowTalents()

	// Blackout and Silence both land a stun/silence on the target; a
	// raid boss, built above core.CharacterMaxLevel, is immune to every
	// CC effect in this sim (see sim/mage/talents.go's Frostbite/Shatter
	// note for the same rule), so neither changes a damage, hit, crit or
	// mana number here. Named, not modeled.
	_, _ = priest.Talents.Blackout, priest.Talents.Silence

	// Shadow Reach is range only, like Mage's Arctic Reach: the mod
	// system has no range kind and adding one would model nothing on a
	// stationary target. Named, not modeled.
	_ = priest.Talents.ShadowReach

	// Improved Psychic Scream shortens the cooldown of a fear effect;
	// Psychic Scream is not registered in this package, and a fear is
	// CC a raid boss is immune to regardless. Named, not modeled.
	_ = priest.Talents.ImprovedPsychicScream

	// Improved Fade shortens Fade's cooldown; Fade is a threat-dump
	// cooldown this package does not register and would not change a
	// damage or mana number if it did. Named, not modeled.
	_ = priest.Talents.ImprovedFade
}

// applyDeclarativeShadowTalents is every Shadow talent that is a pure
// modifier on a set of spells, following sim/mage/talents.go's model:
// config that can be read against a tooltip line by line. Talents with
// state or a proc keep their own function (applySpiritTap,
// applyShadowWeaving, applyDarkness, above).
func (priest *Priest) applyDeclarativeShadowTalents() {
	t := priest.Talents

	if t.TwinDisciplines > 0 {
		// "Increases the damage and healing of your instant cast spells
		// by 1%/2%/3%/4%/5%." The Shadow damage half is Mind Flay,
		// Devouring Plague, Shadow Word: Pain and Shadow Word: Death
		// (PriestSpellMaskInstantShadowDamage); the healing half is the
		// instant heals (PriestSpellMaskInstantHealing).
		//
		// SpellMod_DamageDone_Pct's FloatValue reaches
		// Spell.ApplyMultiplicativeDamageBonus, which does
		// `DamageMultiplier *= multiplier` (sim/core/spell.go,
		// TestApplyDamageBonusHelpers in spell_mod_test.go pins x1.5 ->
		// 1.5): the mod kind's own "+5% = 0.05" doc comment describes
		// SoD's additive accumulator, not this fork's direct-multiplier
		// port (PORTING.md's spell_mod.go section), so the value passed
		// here is a full multiplier, not an offset.
		priest.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_DamageDone_Pct,
			ClassMask:  PriestSpellMaskInstantShadowDamage | PriestSpellMaskInstantHealing,
			FloatValue: 1 + twinDisciplinesDamagePerRank*float64(rankOf("twin_disciplines", t.TwinDisciplines)),
		})
	}

	if t.ImprovedMindFlay > 0 {
		// "Your Mind Flay now deals 10%/20% more damage, gains 5/10
		// yards increased range, but slows the target's movement speed
		// by 35%/20%." The range bonus and the snare change nothing a
		// sim computes against a stationary, in-range target; the
		// damage bonus is the half that moves Shadow DPS.
		priest.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_DamageDone_Pct,
			ClassMask:  PriestSpellMaskMindFlay,
			FloatValue: 1 + improvedMindFlayDamagePerRank*float64(rankOf("improved_mind_flay", t.ImprovedMindFlay)),
		})
	}

	if t.DevouringContagion > 0 {
		// "Reduces the mana cost of your Devouring Plague by 25%/50%."
		// The mana-cost half is modelled directly below; the second
		// half of the tooltip - the DoT spreading to a nearby enemy
		// when its target dies - needs a second target in range and a
		// kill mid-DoT, neither of which this package's reference
		// encounters (single target, no scripted death) ever produce,
		// the same gap Mage's Wake of Fire kill-crit documents. Not
		// modelled; nothing in the standard rotation depends on it.
		priest.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Pct,
			ClassMask: PriestSpellMaskDevouringPlague,
			IntValue:  devouringContagionCostPctPerRank * int64(rankOf("devouring_contagion", t.DevouringContagion)),
		})
	}
}

// rankOf clamps a talent rank to that talent's own max rank, read as the
// length of its generated rank-spell list so it cannot drift from the
// tree - sim/mage/talents.go's rankOf, the same fix for the same cause.
// core.FillTalentsProto does not validate a talent string against the
// client's max rank per node, so an unvalidated or stale string (shadow
// priest's P1Talents here still predates the client talent-tree rewrite;
// SkipAwaitingForeverTalentRewrite in sim/priest/shadow names it) can
// hand a proto field a rank past its talent's real maximum. Caught live
// in this package: with no clamp, P1Talents reads DevouringContagion as
// rank 5 against a max of 2, and devouringContagionCostPctPerRank * 5
// drove Spell.Cost.Multiplier negative, zeroing Devouring Plague's mana
// cost outright (TestDevouringPlagueRank6ResolvesDistinctlyFromRank5ViaGetSpell
// in sim/priest/shadow caught exactly this).
func rankOf(talent string, rank int32) int32 {
	if max := int32(len(TalentSpellIDs[talent])); rank > max {
		return max
	}
	return rank
}

// Forever's rates for five more talents (1.60.1.70291 talents/priest.json):
// Silent Resolve "Reduces the threat generated by your Holy spells by 10%"
// a rank, Shadow Affinity "Reduces the threat generated by your Shadow
// spells by 10%" a rank, Shadow Focus "chance to hit with Shadow spells by
// 1%" a rank, Searing Light "Increases your Holy damage done by 2/5%" and
// Shadowform "increasing your Shadow damage by 10%, reducing the Mana cost
// of all Shadow spells by 50%, increasing the critical strike damage bonus
// of your Shadow spells by 100%, and reducing Physical damage taken by you
// by 15%". Vanilla's 4% (all threat), 8% (all priest spells), 2% (all
// priest spells), 5% a rank and +15% shadow damage are not Forever's.
const (
	silentResolveThreatPerRank    = 0.10
	shadowAffinityThreatPerRank   = 0.10
	shadowFocusHitPerRank         = 1.0
	shadowformDamageBonus         = 0.10
	shadowformManaCostPct         = -50
	shadowformCritDamageBonus     = 1.0
	shadowformPhysicalDamageTaken = 0.85
)

var searingLightDamage = [3]float64{0, 0.02, 0.05}

// isPriestShadowSpell is whether a spell is one of the priest's own Shadow
// spells, the set Shadow Affinity and Shadow Focus name.
func isPriestShadowSpell(spell *core.Spell) bool {
	return spell.Flags.Matches(SpellFlagPriest) && spell.SpellSchool.Matches(core.SpellSchoolShadow)
}

func (priest *Priest) applySilentResolve() {
	if priest.Talents.SilentResolve == 0 {
		return
	}

	priest.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_Threat_Pct,
		School:     core.SpellSchoolHoly,
		FloatValue: 1 - silentResolveThreatPerRank*float64(rankOf("silent_resolve", priest.Talents.SilentResolve)),
	})
}

// Per-rank talent values, all read off the client's own rank
// descriptions for build 1.60.1.70009 (data/builds/1.60.1.70009/
// talents/priest.json). A Forever patch that changes a number changes a
// line here and nothing else.
const (
	// "Increases the damage and healing of your instant cast spells by
	// 1%" per rank.
	twinDisciplinesDamagePerRank = 0.01
	// "Your Mind Flay now deals 10%/20% more damage" - not a flat
	// per-rank multiple beyond rank 2, but 10% a rank happens to hold
	// for both of this talent's two ranks.
	improvedMindFlayDamagePerRank = 0.10
	// "Reduces the mana cost of your Devouring Plague by 25%" per rank.
	// SpellMod_PowerCost_Pct's IntValue is signed percentage points:
	// "-5% = -5".
	devouringContagionCostPctPerRank = -25
)

func (priest *Priest) applyHolySpecialization() {
	if priest.Talents.HolySpecialization == 0 {
		return
	}

	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagPriest) && spell.SpellSchool.Matches(core.SpellSchoolHoly) {
			spell.BonusCritRating += 1 * float64(priest.Talents.HolySpecialization) * core.CritRatingPerCritChance
		}
	})
}

func (priest *Priest) applySearingLight() {
	if priest.Talents.SearingLight == 0 {
		return
	}

	priest.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_PriestSmite || spell.SpellCode == SpellCode_PriestHolyFire {
			spell.DamageMultiplierAdditive += searingLightDamage[rankOf("searing_light", priest.Talents.SearingLight)]
		}
	})
}

func (priest *Priest) applySpiritTap() {
	if priest.Talents.SpiritTap == 0 {
		return
	}

	spellID := []int32{0, 15270, 15335, 15336, 15337, 15338}[priest.Talents.SpiritTap]
	statDep := priest.NewDynamicMultiplyStat(stats.Spirit, 2.0)

	priest.SpiritTapAura = priest.RegisterAura(core.Aura{
		ActionID: core.ActionID{SpellID: spellID},
		Label:    "Spirit Tap",
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			priest.EnableDynamicStatDep(sim, statDep)
			priest.PseudoStats.SpiritRegenRateCasting += 0.50
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			priest.DisableDynamicStatDep(sim, statDep)
			priest.PseudoStats.SpiritRegenRateCasting -= 0.50
		},
	})
}

func (priest *Priest) applyShadowAffinity() {
	if priest.Talents.ShadowAffinity == 0 {
		return
	}

	priest.OnSpellRegistered(func(spell *core.Spell) {
		if isPriestShadowSpell(spell) {
			spell.ThreatMultiplier *= 1 - shadowAffinityThreatPerRank*float64(priest.Talents.ShadowAffinity)
		}
	})
}

func (priest *Priest) applyShadowFocus() {
	if priest.Talents.ShadowFocus == 0 {
		return
	}

	bonusHit := shadowFocusHitPerRank * float64(priest.Talents.ShadowFocus) * core.HitRatingPerHitChance
	priest.OnSpellRegistered(func(spell *core.Spell) {
		if isPriestShadowSpell(spell) {
			spell.BonusHitRating += bonusHit
		}
	})
}

func (priest *Priest) applyShadowWeaving() {
	if priest.Talents.ShadowWeaving == 0 {
		return
	}

	// Forever's Shadow Weaving is a self buff, not vanilla's target
	// debuff: the live talent text (Wowhead overlay, build 1.60.1.70009,
	// three ranks) reads "Your Shadow damage spells have a 33/67/100%
	// chance to increase the Shadow damage you deal by 2% for 15 sec,
	// stacking up to 5 times." So the stacks live on the priest, raise
	// the priest's own Shadow damage, and are 2% a stack where vanilla's
	// debuff (core.ShadowWeavingAura, still available as a raid debuff
	// option) is 3%. Blizzard's 1 October 2026 note "Shadow Weaving can
	// no longer fail to apply" removed the proc's own hit roll; the
	// per-rank chance in the text stays, and is 100% at 3/3.
	rank := int(priest.Talents.ShadowWeaving)
	procChance := shadowWeavingProcChance(rank)
	priest.ShadowWeavingAura = priest.RegisterAura(core.Aura{
		Label:     "Shadow Weaving",
		ActionID:  core.ActionID{SpellID: core.ShadowWeavingSpellIDs[rank]},
		Duration:  shadowWeavingDuration,
		MaxStacks: shadowWeavingMaxStacks,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] /= 1 + shadowWeavingPerStack*float64(oldStacks)
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] *= 1 + shadowWeavingPerStack*float64(newStacks)
		},
	})

	priest.ShadowWeavingProc = priest.GetOrRegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: core.ShadowWeavingSpellIDs[rank]},
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagNoMetrics,
		SpellSchool: core.SpellSchoolShadow,

		// Callers invoke this only on a landed Shadow hit.
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			if procChance < 1 && !sim.Proc(procChance, "Shadow Weaving") {
				return
			}
			priest.ShadowWeavingAura.Activate(sim)
			priest.ShadowWeavingAura.AddStack(sim)
		},
	})
}

// Shadow Weaving's live numbers (talent text, build 1.60.1.70009 with the
// Wowhead overlay): 2% Shadow damage a stack, five stacks, 15 seconds, and
// a 33% chance a rank to add a stack.
const (
	shadowWeavingPerStack  = 0.02
	shadowWeavingMaxStacks = 5
	shadowWeavingDuration  = time.Second * 15
)

func shadowWeavingProcChance(rank int) float64 {
	return min(1, float64(rank)/3)
}

func (priest *Priest) AddShadowWeavingStack(sim *core.Simulation, target *core.Unit) {
	if priest.ShadowWeavingProc == nil {
		return
	}

	priest.ShadowWeavingProc.Cast(sim, target)
}

func (priest *Priest) applyDarkness() {
	if priest.Talents.Darkness == 0 {
		return
	}

	multiplier := 0.02 * float64(priest.Talents.Darkness)

	priest.RegisterAura(core.Aura{
		Label: "Darkness",
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			baseDamageAffectedSpells := core.FilterSlice(
				core.Flatten(
					[][]*core.Spell{
						priest.MindBlast,
						priest.DevouringPlague,
					},
				),
				func(spell *core.Spell) bool { return spell != nil },
			)

			fullDamageAffectedSpells := core.FilterSlice(
				core.Flatten(
					[][]*core.Spell{
						priest.ShadowWordPain,
					},
				),
				func(spell *core.Spell) bool { return spell != nil },
			)

			for _, spells := range priest.MindFlay {
				fullDamageAffectedSpells = append(
					fullDamageAffectedSpells,
					core.FilterSlice(spells, func(spell *core.Spell) bool { return spell != nil })...,
				)
			}

			for _, spell := range baseDamageAffectedSpells {
				spell.BaseDamageMultiplierAdditive += multiplier
			}

			for _, spell := range fullDamageAffectedSpells {
				spell.DamageMultiplierAdditive += multiplier
			}
		},
	})
}

// innerFocusCritRating is Inner Focus's critical bonus: 25%, "if it is a
// non-periodic spell and capable of a critical effect".
var innerFocusCritRating = 25 * float64(core.CritRatingPerCritChance)

// innerFocusAddsCrit is whether Inner Focus's critical bonus reaches a
// spell. Blizzard's 1 October 2026 notes ("Inner Focus's crit bonus no
// longer applies to periodic effects") and the live text keep it off pure
// damage-over-time spells and channels, whose critical strikes are
// periodic - which matters now that Devouring Plague's ticks can crit.
func innerFocusAddsCrit(spell *core.Spell) bool {
	return !spell.Flags.Matches(core.SpellFlagPureDot | core.SpellFlagChanneled)
}

func (priest *Priest) registerInnerFocus() {
	if !priest.Talents.InnerFocus {
		return
	}

	actionID := core.ActionID{SpellID: 14751}

	priest.InnerFocusAura = priest.RegisterAura(core.Aura{
		Label:    "Inner Focus",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range priest.Spellbook {
				if spell.Flags.Matches(SpellFlagPriest) && spell.Cost != nil {
					spell.Cost.Multiplier -= 100
					if innerFocusAddsCrit(spell) {
						spell.BonusCritRating += innerFocusCritRating
					}
				}
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range priest.Spellbook {
				if spell.Flags.Matches(SpellFlagPriest) && spell.Cost != nil {
					spell.Cost.Multiplier += 100
					if innerFocusAddsCrit(spell) {
						spell.BonusCritRating -= innerFocusCritRating
					}
				}
			}
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.Flags.Matches(SpellFlagPriest) {
				// Remove the buff and put skill on CD
				aura.Deactivate(sim)
				priest.InnerFocus.CD.Use(sim)
				priest.UpdateMajorCooldowns()
			}
		},
	})

	priest.InnerFocus = priest.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL | core.SpellFlagHelpful,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    priest.NewTimer(),
				Duration: time.Minute * 3,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			priest.InnerFocusAura.Activate(sim)
		},
	})

	priest.AddMajorCooldown(core.MajorCooldown{
		Spell: priest.InnerFocus,
		Type:  core.CooldownTypeDPS,
	})
}

func (priest *Priest) registerShadowform() {
	if !priest.Talents.Shadowform {
		return
	}

	actionID := core.ActionID{SpellID: 15473}

	shadowformMods := []*core.SpellMod{
		priest.AddDynamicMod(core.SpellModConfig{
			Kind:            core.SpellMod_CritDamageBonus_Flat,
			School:          core.SpellSchoolShadow,
			ClassSpellsOnly: true,
			FloatValue:      shadowformCritDamageBonus,
		}),
		priest.AddDynamicMod(core.SpellModConfig{
			Kind:            core.SpellMod_PowerCost_Pct,
			School:          core.SpellSchoolShadow,
			ClassSpellsOnly: true,
			IntValue:        shadowformManaCostPct,
		}),
	}

	priest.ShadowformAura = priest.RegisterAura(core.Aura{
		Label:    "Shadowform",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] *= 1 + shadowformDamageBonus
			aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexPhysical] *= shadowformPhysicalDamageTaken
			core.Each(shadowformMods, (*core.SpellMod).Activate)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow] /= 1 + shadowformDamageBonus
			aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexPhysical] /= shadowformPhysicalDamageTaken
			core.Each(shadowformMods, (*core.SpellMod).Deactivate)
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.SpellSchool.Matches(core.SpellSchoolHoly) {
				aura.Deactivate(sim)
			}
		},
	})

	priest.Shadowform = priest.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,

		// 40% of base mana: the client prices the form through
		// SpellPower.PowerCostPct (ShadowformManaCostPct), not the flat
		// cost column, which reads 0.
		ManaCost: core.ManaCostOptions{
			BaseCost: ShadowformManaCostPct[0] / 100,
		},

		// Shadowform is cast like any other instant: the client's own
		// ShadowformCooldownMS (constants_auto_gen.go) carries 1500ms for
		// both the GCD and a same-length cooldown, not the GCD-less,
		// cooldown-less toggle this previously modeled.
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    priest.NewTimer(),
				Duration: time.Millisecond * time.Duration(ShadowformCooldownMS[0]),
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			priest.ShadowformAura.Activate(sim)
		},
	})
}
