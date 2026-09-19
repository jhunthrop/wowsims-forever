package mage

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// FOREVER: this file is written against the CLIENT's trait trees for
// build 1.60.1.69893, not vanilla's, and every number below is the one
// the client's own rank description gives. Where Forever changed a
// talent's meaning rather than its value - the resist-reduction talents
// that are now hit talents, research/08-stats.md section 1.2 - the
// talent was rebuilt rather than renamed.
//
// Talents that are a pure modifier on a set of spells live in
// applyDeclarativeTalents as SpellMod config, so they can be read
// against a tooltip line by line and a weekly number change is a
// one-line edit. Talents with state, a timer or a proc keep their own
// function.
//
// A talent the engine does not model yet is named with the reason
// rather than dropped, so the reader can tell "not implemented" from
// "not noticed".

// ForeverFrostTalents is the reference build the regression suite runs.
// It is not advice; it is a fixed input so a DPS change is attributable
// to the engine rather than to a build edit.
//
// Written against the CLIENT's tree, so its segments are 18, 17 and 19
// characters (TalentTreeSizes), in the client's tree order Arcane, Fire,
// Frost. Spend:
//
//	Arcane 20: Arcane Focus 5, Arcane Concentration 5, Arcane Geometry 2,
//	           Arcane Impact 3, Arcane Shielding 1, Arcane Meditation 3,
//	           Missile Barrage 1
//	Fire 0
//	Frost 31: Improved Frostbolt 5, Ice Shards 5, Piercing Ice 3,
//	          Frost Channeling 1, Ice Lance 1, Arctic Reach 2, Shatter 3,
//	          Improved Cone of Cold 2, Cold Snap 1, Fingers of Frost 2,
//	          Winter's Chill 5, Ice Barrier 1
//
// 51 points, every tier gate and prerequisite satisfied, and Ice Barrier
// at the bottom of Frost is what the 31 points buy.
const ForeverFrostTalents = "050005023010310000-00000000000000000-0505000311020321251"

func (mage *Mage) ApplyTalents() {
	mage.applyDeclarativeTalents()
	mage.applyArcaneTalents()
	mage.applyFireTalents()
	mage.applyFrostTalents()
}

func (mage *Mage) applyArcaneTalents() {
	mage.applyArcaneConcentration()
	mage.registerPresenceOfMindCD()
	mage.registerArcanePowerCD()

	// Magic Absorption: "Increases all your resistances by 5" per rank.
	// The mana returned on a full resist is not modelled: nothing in the
	// sim resists a boss-cast spell against the player.
	if mage.Talents.MagicAbsorption > 0 {
		mage.AddResistances(magicAbsorptionResistancePerRank * float64(mage.Talents.MagicAbsorption))
	}

	// Arcane Meditation: "Allows 17%/33%/50% of your Mana regeneration
	// to continue while casting." Not 5% a rank, which was vanilla's.
	mage.PseudoStats.SpiritRegenRateCasting += arcaneMeditationRegenWhileCasting[rankIndex(mage.Talents.ArcaneMeditation, arcaneMeditationRegenWhileCasting[:])]

	// Arcane Mind: "Increases your Intellect by 2%" per rank. Vanilla's
	// raised Mana directly; Forever's raises Intellect, which reaches
	// Mana through the stat dependency and also reaches spell crit. The
	// crit-damage half is in applyDeclarativeTalents.
	if mage.Talents.ArcaneMind > 0 {
		mage.MultiplyStat(stats.Intellect, 1.0+arcaneMindIntellectPerRank*float64(mage.Talents.ArcaneMind))
	}

	// Wand Specialization increases wand damage; this sim does not
	// model wand casts.
	_ = mage.Talents.WandSpecialization

	// Arcane Blast is a bool talent that grants a new cast of Arcane
	// Blast (MageSpellMaskArcaneBlast exists and Incineration targets
	// it), but this package does not register the spell yet.
	_ = mage.Talents.ArcaneBlast

	// Arcane Shielding is Improved Mana Shield, and the reference build
	// (ForeverFrostTalents) spends its point here; Mana Shield itself is
	// not registered in this package, so there is nothing for it to
	// improve.
	_ = mage.Talents.ArcaneShielding

	// Arcane Geometry and Improved Counterspell are range and a silence:
	// the sim models neither, and no mod kind would model them.
	_, _ = mage.Talents.ArcaneGeometry, mage.Talents.ImprovedCounterspell

	// Improved Channeling is pushback avoidance and Arcane Resilience is
	// armour from Intellect; neither changes a damage or mana number on
	// a Patchwerk fight.
	_, _ = mage.Talents.ImprovedChanneling, mage.Talents.ArcaneResilience

	// Missile Barrage would need the Arcane Missiles channel to be
	// re-costed and re-timed mid-fight, which this package's Arcane
	// Missiles does not support yet. Inert, not forgotten.
	_ = mage.Talents.MissileBarrage
}

func (mage *Mage) applyFireTalents() {
	mage.applyIgnite()
	mage.applyImprovedScorch()
	mage.applyMasterOfElements()

	mage.registerCombustionCD()

	// Flame Throwing is range, Impact is a stun, Improved Fire Ward is a
	// reflect: none of the three changes a damage number here.
	_, _, _ = mage.Talents.FlameThrowing, mage.Talents.Impact, mage.Talents.ImprovedFireWard

	// Hot Streak would need Pyroblast's cast time to change on a
	// stacking buff driven by non-periodic Fire crits. Inert until
	// Pyroblast carries a dynamic cast-time mod.
	_ = mage.Talents.HotStreak
}

func (mage *Mage) applyFrostTalents() {
	// The client makes Ice Lance and Cold Snap Frost TALENTS (nodes
	// 105767 and 105766), not baseline abilities, so both are registered
	// under their talent rather than from RegisterMage: an Arcane mage
	// that never spent the point must not have the spell, or its
	// .results golden moves for a spell it cannot cast.
	mage.registerIceLanceSpell()
	mage.registerColdSnapSpell()
	mage.registerIceBarrierSpell()
	mage.applyWintersChill()

	// Frost Warding, Permafrost and Improved Blizzard are an armour
	// bonus, a snare duration and a snare: Improved Blizzard's chill is
	// applied in blizzard.go, the other two change nothing on a
	// stationary target.
	_, _ = mage.Talents.FrostWarding, mage.Talents.Permafrost

	// Arctic Reach is range and radius. The mod system has no range kind
	// and adding one would model nothing.
	_ = mage.Talents.ArcticReach

	// Ice Block is a defensive immunity the rotation never casts.
	_ = mage.Talents.IceBlock

	// Frostbite, Shatter and Fingers of Frost all turn on a target being
	// Frozen. Nothing in this sim freezes a raid boss - Frost Nova and
	// Frostbite are not registered, and a boss is immune to both - so
	// all three are inert here rather than guessed. isTargetFrozen in
	// ice_lance.go is the single place that would start returning true.
	_, _, _ = mage.Talents.Frostbite, mage.Talents.Shatter, mage.Talents.FingersOfFrost
}

func (mage *Mage) applyArcaneConcentration() {
	if mage.Talents.ArcaneConcentration == 0 {
		return
	}

	procChance := 0.02 * float64(mage.Talents.ArcaneConcentration)

	mage.ClearcastingAura = mage.RegisterAura(core.Aura{
		Label:    "Clearcasting",
		ActionID: core.ActionID{SpellID: 12577},
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolCostMultiplier.AddToMagicSchools(-100)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolCostMultiplier.AddToMagicSchools(100)
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			// OnCastComplete is called after OnSpellHitDealt / etc, so don't deactivate if it was just activated.
			if aura.RemainingDuration(sim) == aura.Duration {
				return
			}
			if !spell.Flags.Matches(SpellFlagMage) {
				return
			}
			if spell.Cost != nil && spell.Cost.GetCurrentCost() == 0 {
				return
			}
			aura.Deactivate(sim)
		},
	})

	mage.RegisterAura(core.Aura{
		Label:    "Arcane Concentration",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.Flags.Matches(SpellFlagMage) || spell.SpellCode == SpellCode_MageArcaneMissiles {
				return
			}

			// TODO: Classic verify arcane missile proc chance
			// Arcane Missile ticks can proc CC, just at a low rate of about 1.5% with 5/5 Arcane Concentration
			// if spell == mage.ArcaneMissilesTickSpell {
			// 	procChance *= 0.15
			// }

			if sim.Proc(procChance, "Arcane Concentration") {
				mage.ClearcastingAura.Activate(sim)
			}
		},
	})
}

func (mage *Mage) registerPresenceOfMindCD() {
	if !mage.Talents.PresenceOfMind {
		return
	}

	actionID := core.ActionID{SpellID: 12043}
	cooldown := time.Second * 180

	affectedSpells := []*core.Spell{}
	pomAura := mage.RegisterAura(core.Aura{
		Label:    "Presence of Mind",
		ActionID: actionID,
		Duration: time.Second * 15,
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			for spellIdx := range mage.Spellbook {
				if spell := mage.Spellbook[spellIdx]; spell.DefaultCast.CastTime > 0 {
					affectedSpells = append(affectedSpells, spell)
				}
			}
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			core.Each(affectedSpells, func(spell *core.Spell) {
				spell.CastTimeMultiplier -= 1
			})
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			core.Each(affectedSpells, func(spell *core.Spell) {
				spell.CastTimeMultiplier += 1
			})
			mage.PresenceOfMind.CD.Use(sim)
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if !slices.Contains(affectedSpells, spell) {
				return
			}

			aura.Deactivate(sim)
		},
	})

	mage.PresenceOfMind = mage.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete,
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    mage.NewTimer(),
				Duration: cooldown,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return mage.GCD.IsReady(sim)
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			pomAura.Activate(sim)
		},
	})

	mage.AddMajorCooldown(core.MajorCooldown{
		Spell: mage.PresenceOfMind,
		Type:  core.CooldownTypeDPS,
	})
}

func (mage *Mage) registerArcanePowerCD() {
	if !mage.Talents.ArcanePower {
		return
	}

	actionID := core.ActionID{SpellID: 12042}

	affectedSpells := []*core.Spell{}

	mage.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagMage) {
			affectedSpells = append(affectedSpells, spell)
		}
	})

	mage.ArcanePowerAura = mage.RegisterAura(core.Aura{
		Label:    "Arcane Power",
		ActionID: actionID,
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.DamageMultiplierAdditive += 0.3
				if spell.Cost != nil {
					spell.Cost.Multiplier += 30
				}
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range affectedSpells {
				spell.DamageMultiplierAdditive -= 0.3
				if spell.Cost != nil {
					spell.Cost.Multiplier -= 30
				}
			}
		},
	})
	core.RegisterPercentDamageModifierEffect(mage.ArcanePowerAura, 1.3)

	spell := mage.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete,
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    mage.NewTimer(),
				Duration: time.Second * 180,
			},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			mage.ArcanePowerAura.Activate(sim)
		},
	})

	mage.AddMajorCooldown(core.MajorCooldown{
		Spell: spell,
		Type:  core.CooldownTypeDPS,
	})
}

func (mage *Mage) applyImprovedScorch() {
	if mage.Talents.ImprovedScorch == 0 {
		return
	}

	mage.ImprovedScorchAuras = mage.NewEnemyAuraArray(func(unit *core.Unit) *core.Aura {
		return core.ImprovedScorchAura(unit)
	})
}

func (mage *Mage) applyMasterOfElements() {
	if mage.Talents.MasterOfElements == 0 {
		return
	}

	refundCoeff := 0.1 * float64(mage.Talents.MasterOfElements)
	manaMetrics := mage.NewManaMetrics(core.ActionID{SpellID: 29076})

	mage.RegisterAura(core.Aura{
		Label:    "Master of Elements",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// "Your Fire and Frost critical strikes will refund 10% of
			// their base mana cost." Forever names the two schools;
			// vanilla's refunded on any crit.
			if !spell.SpellSchool.Matches(core.SpellSchoolFire | core.SpellSchoolFrost) {
				return
			}
			if spell.ProcMask.Matches(core.ProcMaskMeleeOrRanged) {
				return
			}
			if spell.CurCast.Cost == 0 {
				return
			}
			if result.DidCrit() {
				mage.AddMana(sim, spell.Cost.BaseCost*refundCoeff, manaMetrics)
			}
		},
	})
}

func (mage *Mage) registerCombustionCD() {
	if !mage.Talents.Combustion {
		return
	}

	actionID := core.ActionID{SpellID: 11129}
	cd := core.Cooldown{
		Timer:    mage.NewTimer(),
		Duration: time.Minute * 3,
	}

	var fireSpells []*core.Spell
	mage.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellSchool.Matches(core.SpellSchoolFire) && spell.Flags.Matches(SpellFlagMage) {
			fireSpells = append(fireSpells, spell)
		}
	})

	numCrits := 0
	critPerStack := 10.0 * core.CritRatingPerCritChance

	mage.CombustionAura = mage.RegisterAura(core.Aura{
		Label:     "Combustion",
		ActionID:  actionID,
		Duration:  core.NeverExpires,
		MaxStacks: 20,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			numCrits = 0
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			cd.Use(sim)
			mage.UpdateMajorCooldowns()
		},
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
			bonusCrit := critPerStack * float64(newStacks-oldStacks)
			for _, spell := range fireSpells {
				spell.BonusCritRating += bonusCrit
			}
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || numCrits >= 3 || !spell.SpellSchool.Matches(core.SpellSchoolFire) || !spell.Flags.Matches(SpellFlagMage) {
				return
			}

			// Ignite, Living Bomb explosions, and Fire Blast with Overheart don't consume crit stacks
			// To Do: Classic - I don't believe ignite can crit so can probably remove this check?
			if spell.SpellCode == SpellCode_MageIgnite {
				return
			}

			// TODO: This wont work properly with flamestrike
			aura.AddStack(sim)

			if result.DidCrit() {
				numCrits++
				if numCrits == 3 {
					aura.Deactivate(sim)
				}
			}
		},
	})

	spell := mage.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete,
		Cast: core.CastConfig{
			CD: cd,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return !mage.CombustionAura.IsActive()
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			mage.CombustionAura.Activate(sim)
			mage.CombustionAura.AddStack(sim)
		},
	})

	mage.AddMajorCooldown(core.MajorCooldown{
		Spell: spell,
		Type:  core.CooldownTypeDPS,
	})
}

func (mage *Mage) applyWintersChill() {
	if mage.Talents.WintersChill == 0 {
		return
	}

	procChance := float64(mage.Talents.WintersChill) * 0.2

	wcAuras := mage.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.WintersChillAura(target)
	})
	mage.Env.RegisterPreFinalizeEffect(func() {
		for _, spell := range mage.GetSpellsMatchingSchool(core.SpellSchoolFrost) {
			spell.RelatedAuras = append(spell.RelatedAuras, wcAuras)
		}
	})

	mage.RegisterAura(core.Aura{
		Label:    "Winters Chill Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.SpellSchool.Matches(core.SpellSchoolFrost) {
				return
			}

			if sim.Proc(procChance, "Winters Chill") {
				aura := wcAuras.Get(result.Target)
				aura.Activate(sim)
				if aura.IsActive() {
					aura.AddStack(sim)
				}
			}
		},
	})
}

// Per-rank talent values, all read off the client's own rank
// descriptions for build 1.60.1.69893 (data/builds/1.60.1.69893/
// talents/mage.json). A Forever patch that changes a number changes a
// line here and nothing else.
const (
	// "Reduces your target's resistance ... and reduces the threat
	// caused by your Arcane spells by 15%." Vanilla's was 20%.
	arcaneSubtletyThreatReductionPerRank = 0.15
	// "Improves your chance to hit with Arcane spells by 1%." In
	// vanilla this talent reduced the target's chance to resist
	// instead; Forever folded that lever into unified hit
	// (research/08-stats.md section 1.2), so it is 1% a rank of hit,
	// not 2% a rank of resist reduction.
	arcaneFocusHitPerRank = 1.0
	// "Increases the critical strike chance of your Arcane spells by
	// 2%." New: vanilla's tree had no Arcane crit talent here.
	arcaneImpactCritPerRank = 2.0
	// "Increases your Intellect by 2% and increases the critical strike
	// damage bonus of your Arcane spells by 20%."
	arcaneMindIntellectPerRank  = 0.02
	arcaneMindCritDamagePerRank = 0.20
	// "Increases the damage done by your spells by 1% and your critical
	// strike chance by 1%."
	arcaneInstabilityDamagePerRank = 1
	arcaneInstabilityCritPerRank   = 1.0
	// "Increases all your resistances by 5." Vanilla's gave 2.
	magicAbsorptionResistancePerRank = 5.0

	// "Gives your Fire spells a ...% chance to not lose casting time
	// when you take damage and reduces the threat caused by your Fire
	// spells by 10%." Vanilla's was 15% a rank.
	burningSoulThreatReductionPerRank = 0.10
	// "Increases the critical strike chance of your Fire spells by 2%."
	criticalMassCritPerRank = 2.0
	// "Increases the damage done by your Fire spells by 2%."
	firePowerDamagePerRank = 2
	// "Increases the critical strike chance of your Fire Blast, Ice
	// Lance, Arcane Blast, and Scorch spells by 2%." The spell list is
	// Forever's and crosses all three schools, which is exactly what a
	// mask expresses and a school check cannot.
	incinerationCritPerRank = 2.0
	// "Increases the critical strike chance of your Flamestrike spell
	// by 5%."
	improvedFlamestrikeCritPerRank = 5.0
	// "Reduces the casting time of your Fireball and Frostfire Bolt
	// spells by 0.1 sec."
	improvedFireballCastTimePerRank = -100 * time.Millisecond
	// "Reduces the cooldown of your Fire Blast spell by 1 sec." The
	// kill-triggered crit bonus on the same talent needs a killing blow,
	// which a Patchwerk fight never produces.
	wakeOfFireCooldownPerRank = -1 * time.Second

	// "Reduces the casting time of your Frostbolt spell by 0.1 sec."
	improvedFrostboltCastTimePerRank = -100 * time.Millisecond
	// "Improves your chance to hit with Frost and Fire spells by 1%."
	// Vanilla's reduced resist chance by 2% a rank; see Arcane Focus.
	elementalPrecisionHitPerRank = 1.0
	// "Increases the critical strike damage bonus of your Frost spells
	// by 20%."
	iceShardsCritDamagePerRank = 0.20
	// "Increases the damage done by your Frost spells by 2%."
	piercingIceDamagePerRank = 2
	// "Reduces the mana cost of your Frost spells by 5% and reduces the
	// threat caused by your Frost spells by 10%."
	frostChannelingCostPerRank            = -5
	frostChannelingThreatReductionPerRank = 0.10
	// "Reduces the cooldown of your Frost Nova spell by 2 sec."
	improvedFrostNovaCooldownPerRank = -2 * time.Second
)

// Talents whose ranks do not scale linearly need the client's table
// rather than a multiplication.
var (
	// "Allows 17%/33%/50% of your Mana regeneration to continue while
	// casting." Vanilla's was a flat 5% a rank.
	arcaneMeditationRegenWhileCasting = [4]float64{0, 0.17, 0.33, 0.50}
	// "Increases the damage dealt by your Cone of Cold spell by
	// 12%/23%/35%."
	improvedConeOfColdDamage = [4]int64{0, 12, 23, 35}
)

// rankIndex clamps a talent rank to a lookup table's highest index.
// core.FillTalentsProto does not validate a talent string against the
// client's max rank per node, so a string with more points in a talent
// than the talent allows (or a corrupt/hand-edited one) would otherwise
// index one of the tables above out of range instead of reading the
// talent's max-rank value.
func rankIndex[T any](rank int32, table []T) int {
	if i := int(rank); i >= 0 && i < len(table) {
		return i
	}
	return len(table) - 1
}

// applyDeclarativeTalents is every talent that is a modifier on a set of
// spells. As config these can be read against a tooltip line by line,
// which is what Forever's weekly number changes need; talents with
// state or a timer keep their own functions.
func (mage *Mage) applyDeclarativeTalents() {
	t := mage.Talents

	// ---- Arcane ----

	if t.ArcaneSubtlety > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_Threat_Pct,
			ClassMask:  MageSpellMaskArcaneDamage,
			FloatValue: 1 - arcaneSubtletyThreatReductionPerRank*float64(t.ArcaneSubtlety),
		})
	}

	if t.ArcaneFocus > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_BonusHit_Flat,
			ClassMask:  MageSpellMaskArcaneDamage,
			FloatValue: arcaneFocusHitPerRank * float64(t.ArcaneFocus) * core.HitRatingPerHitChance,
		})
	}

	if t.ArcaneImpact > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_BonusCrit_Flat,
			ClassMask:  MageSpellMaskArcaneDamage,
			FloatValue: arcaneImpactCritPerRank * float64(t.ArcaneImpact) * core.CritRatingPerCritChance,
		})
	}

	if t.ArcaneMind > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_CritDamageBonus_Flat,
			ClassMask:  MageSpellMaskArcaneDamage,
			FloatValue: arcaneMindCritDamagePerRank * float64(t.ArcaneMind),
		})
	}

	if t.ArcaneInstability > 0 {
		// "your spells", with no school named, so every spell this
		// class registers rather than a named set.
		mage.AddStaticMod(core.SpellModConfig{
			Kind:            core.SpellMod_DamageDone_Flat,
			ClassSpellsOnly: true,
			IntValue:        arcaneInstabilityDamagePerRank * int64(t.ArcaneInstability),
		})
		mage.AddStaticMod(core.SpellModConfig{
			Kind:            core.SpellMod_BonusCrit_Flat,
			ClassSpellsOnly: true,
			FloatValue:      arcaneInstabilityCritPerRank * float64(t.ArcaneInstability) * core.CritRatingPerCritChance,
		})
	}

	// ---- Fire ----

	if t.BurningSoul > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_Threat_Pct,
			ClassMask:  MageSpellMaskFireDamage,
			FloatValue: 1 - burningSoulThreatReductionPerRank*float64(t.BurningSoul),
		})
	}

	if t.CriticalMass > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_BonusCrit_Flat,
			ClassMask:  MageSpellMaskFireDamage,
			FloatValue: criticalMassCritPerRank * float64(t.CriticalMass) * core.CritRatingPerCritChance,
		})
	}

	if t.FirePower > 0 {
		// Ignite is deliberately outside MageSpellMaskFireDamage: it is
		// a consequence of a Fire crit, not a Fire spell the mage casts.
		mage.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_DamageDone_Flat,
			ClassMask: MageSpellMaskFireDamage,
			IntValue:  firePowerDamagePerRank * int64(t.FirePower),
		})
	}

	if t.Incineration > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_BonusCrit_Flat,
			ClassMask:  MageSpellMaskFireBlast | MageSpellMaskIceLance | MageSpellMaskArcaneBlast | MageSpellMaskScorch,
			FloatValue: incinerationCritPerRank * float64(t.Incineration) * core.CritRatingPerCritChance,
		})
	}

	if t.ImprovedFlamestrike > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_BonusCrit_Flat,
			ClassMask:  MageSpellMaskFlamestrike,
			FloatValue: improvedFlamestrikeCritPerRank * float64(t.ImprovedFlamestrike) * core.CritRatingPerCritChance,
		})
	}

	if t.ImprovedFireball > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_CastTime_Flat,
			ClassMask: MageSpellMaskFireball | MageSpellMaskFrostfireBolt,
			TimeValue: improvedFireballCastTimePerRank * time.Duration(t.ImprovedFireball),
		})
	}

	if t.WakeOfFire > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_Cooldown_Flat,
			ClassMask: MageSpellMaskFireBlast,
			TimeValue: wakeOfFireCooldownPerRank * time.Duration(t.WakeOfFire),
		})
	}

	// ---- Frost ----

	if t.ImprovedFrostbolt > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_CastTime_Flat,
			ClassMask: MageSpellMaskFrostbolt,
			TimeValue: improvedFrostboltCastTimePerRank * time.Duration(t.ImprovedFrostbolt),
		})
	}

	if t.ElementalPrecision > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_BonusHit_Flat,
			ClassMask:  MageSpellMaskFrostDamage | MageSpellMaskFireDamage,
			FloatValue: elementalPrecisionHitPerRank * float64(t.ElementalPrecision) * core.HitRatingPerHitChance,
		})
	}

	if t.IceShards > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_CritDamageBonus_Flat,
			ClassMask:  MageSpellMaskFrostDamage,
			FloatValue: iceShardsCritDamagePerRank * float64(t.IceShards),
		})
	}

	if t.PiercingIce > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_DamageDone_Flat,
			ClassMask: MageSpellMaskFrostDamage,
			IntValue:  piercingIceDamagePerRank * int64(t.PiercingIce),
		})
	}

	if t.FrostChanneling > 0 {
		// The mana half reads "your Frost spells", which includes Ice
		// Barrier; the threat half is the same set.
		mage.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Pct,
			ClassMask: MageSpellMaskFrost,
			IntValue:  frostChannelingCostPerRank * int64(t.FrostChanneling),
		})
		mage.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_Threat_Pct,
			ClassMask:  MageSpellMaskFrost,
			FloatValue: 1 - frostChannelingThreatReductionPerRank*float64(t.FrostChanneling),
		})
	}

	if t.ImprovedConeOfCold > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_DamageDone_Flat,
			ClassMask: MageSpellMaskConeOfCold,
			IntValue:  improvedConeOfColdDamage[rankIndex(t.ImprovedConeOfCold, improvedConeOfColdDamage[:])],
		})
	}

	if t.ImprovedFrostNova > 0 {
		mage.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_Cooldown_Flat,
			ClassMask: MageSpellMaskFrostNova,
			TimeValue: improvedFrostNovaCooldownPerRank * time.Duration(t.ImprovedFrostNova),
		})
	}
}
