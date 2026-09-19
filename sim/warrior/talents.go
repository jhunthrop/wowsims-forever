package warrior

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// FOREVER: this file is written against the client's own trait tables,
// build 1.60.1.69893, as Task 17 generated them into talents_auto_gen.go
// and proto.WarriorTalents. Every number below is the number the
// client's rank description gives, not vanilla's: thirteen of the
// warrior's talents changed meaning between the two, four of them from
// a damage or resistance effect to a hit or rage one. Where a
// description leaves a figure unresolved (the client's own text has a
// `$h` in it) the site is marked `unconfirmed` and the validation job's
// comparison is what clears it.
//
// Talents that are pure modifiers on a set of spells live in
// applyDeclarativeTalents as core.SpellModConfig, because config can be
// read against a tooltip line by line and Forever changes these numbers
// weekly. Talents with state or a timer keep their own function.

// ForeverFuryTalents is the reference build the regression suite runs: a
// deep Fury build reaching Bloodthirst, the 31-point Fury talent, with
// the Arms points where a Fury warrior actually spends them. It is not
// advice; it is a fixed input, so a DPS change is attributable to the
// engine rather than to a build edit.
//
// It is written against the CLIENT's tree, so its three segments are 17,
// 18 and 18 characters (TalentTreeSizes), not vanilla's 18/19/16.
// Spend, against the client's own tree:
//
//	Arms 20: Improved Heroic Strike 3, Deflection 3, Improved Rend 3,
//	         Improved Tactical Mastery 5, Anger Management 1,
//	         Deep Wounds 3, Impale 2
//	Fury 31: Booming Voice 1, Cruelty 5, Unbridled Wrath 5,
//	         Improved Cleave 3, Piercing Howl 1, Enrage 5,
//	         Improved Execute 1, Precision 3, Death Wish 1, Flurry 5,
//	         Bloodthirst 1
//	Protection 0
//
// Improved Rend is 3, not the 2 an earlier draft spent, because the
// client makes Deep Wounds require Improved Rend at rank 3; and the Fury
// half is shaped by the tier gates rather than by taste, because
// reaching Bloodthirst with exactly 31 points forces 25 of them into
// tiers 0-4. TestForeverFuryTalentsAreAValidBuild checks the widths and
// the spend, so a segment one character short fails rather than
// silently reading the neighbouring talent.
const ForeverFuryTalents = "33305013002000000-150531000051310051-000000000000000000"

// ForeverProtectionTalents is the same fixed input for the tank spec.
// The spec's own talent behaviour is still vanilla's - Improved Revenge,
// Defiance, Bastion, Focused Rage, Master of Defense, Improved
// Bloodrage, Improved Shield Wall, Improved Thunder Clap and
// Anticipation all changed meaning in the client's tree and only
// Anticipation and Improved Thunder Clap are rewritten here - so
// sim/warrior/tank_warrior stays skipped until the warrior-protection
// spec is brought up. The string is declared now, and checked by
// TestTheReferenceBuildsRespectTheTiersAndPrerequisites, because the
// one that stood in the tank test was vanilla-shaped (11/2/17) and
// could not be parsed against the client's trees at all.
//
//	Arms 20:       Improved Heroic Strike 3, Deflection 5,
//	               Improved Rend 3, Improved Tactical Mastery 5,
//	               Anger Management 1, Deep Wounds 3
//	Protection 31: Shield Specialization 5, Anticipation 5,
//	               Improved Bloodrage 2, Toughness 5,
//	               Improved Thunder Clap 3, Last Stand 1, Defiance 3,
//	               Improved Sunder Armor 3, Concussion Blow 1,
//	               Bastion 2, Shield Slam 1
const ForeverProtectionTalents = "35305013000000000-000000000000000000-552531003300010201"

// fillWarriorTalents parses a talent string into the proto, positionally
// against TalentTreeSizes. It is the one place that pairing happens, so
// a tree-size change cannot be applied in one caller and missed in
// another.
func fillWarriorTalents(talents *proto.WarriorTalents, s string) {
	core.FillTalentsProto(talents.ProtoReflect(), s, TalentTreeSizes)
}

// ToughnessArmorMultiplier is Toughness: "Increases your Armor value
// from items by 10%" at rank 5, so 2% a point.
func (warrior *Warrior) ToughnessArmorMultiplier() float64 {
	return 1.0 + 0.02*float64(warrior.Talents.Toughness)
}

func (warrior *Warrior) ApplyTalents() {
	// Flat stats. Forever has no combat ratings, so a percentage is the
	// stat: CritRatingPerCritChance and HitRatingPerHitChance are both 1.
	//
	//	Cruelty:      "+5% melee critical strike" at rank 5.
	//	Precision:    "+3% chance to hit" at rank 3. Precision is one of
	//	              the four talents research/08-stats.md 1.2 records as
	//	              converting from resistance reduction to hit; it is a
	//	              new talent here, not a renamed one.
	//	Anticipation: "+20 Defense Skill" at rank 5, so 4 a point - twice
	//	              vanilla's 2, which is what the old body applied.
	//	Deflection:   "+5% Parry" at rank 5.
	warrior.AddStat(stats.Crit, core.CritRatingPerCritChance*1*float64(warrior.Talents.Cruelty))
	warrior.AddStat(stats.Hit, core.HitRatingPerHitChance*1*float64(warrior.Talents.Precision))
	warrior.ApplyEquipScaling(stats.Armor, warrior.ToughnessArmorMultiplier())
	warrior.AddStat(stats.Defense, 4*float64(warrior.Talents.Anticipation))
	warrior.AddStat(stats.Parry, 1*float64(warrior.Talents.Deflection))

	warrior.applyDeclarativeTalents()

	// Talents with real mechanics keep their own functions.
	warrior.applyAngerManagement()
	warrior.applyDeepWounds()
	warrior.applyWeaponmaster()
	warrior.applyUnbridledWrath()
	warrior.applyDualWieldSpecialization()
	warrior.applyEnrage()
	warrior.applyFlurry()
	warrior.applyShieldSpecialization()
	warrior.registerDeathWishCD()
	warrior.registerSweepingStrikesCD()
	warrior.registerLastStandCD()
}

// Talents whose ranks do not scale linearly, or whose values are the
// client's own per-rank figures, need a table rather than a
// multiplication. They live here, in one block, so rankIndex below is
// the only way any of them is read; each is named at the site that
// reads it, which may be another file in this package.
var (
	// Improved Bloodrage: "+5 Rage instantly" at rank 2, so the ranks
	// are 2 and 5 on top of Bloodrage's own 10.
	improvedBloodrageInstantRage = [3]float64{0, 2, 5}
	// Improved Execute: "by 3" at rank 1 and "by 5" at rank 2 - not
	// vanilla's 2 and 5, which is what the old inline table said.
	improvedExecuteRageReduction = [3]int64{0, 3, 5}
	// Improved Rend: "Increases the damage of your Rend ability by
	// 35%" at rank 3.
	improvedRendDamageMultiplier = [4]float64{1, 1.12, 1.23, 1.35}
	// Improved Shield Wall: "+5 sec" at rank 2.
	improvedShieldWallDuration = [3]float64{0, 3, 5}
	// Defiance: "+15% threat caused in Defensive Stance" at rank 5.
	defianceThreatMultiplier = [6]float64{1, 1.03, 1.06, 1.09, 1.12, 1.15}
	// Flurry: "+25% melee attack speed for your next swings" at rank 5,
	// so 5% a point. See makeFlurryAura.
	flurryAttackSpeed = [6]float64{1, 1.05, 1.10, 1.15, 1.20, 1.25}
	// Deep Wounds has a distinct rank spell per rank, unlike the ranks
	// in TalentSpellIDs, which the client reissued as one id.
	deepWoundsSpellIDs = [4]int32{0, 12834, 12849, 12867}
)

// rankIndex clamps a talent rank to a lookup table's highest index.
// core.FillTalentsProto does not validate a talent string against the
// client's max rank per node, so a string with more points in a talent
// than the talent allows (a corrupt or hand-edited one, and the string
// arrives from the web) would otherwise index one of this package's
// rank tables out of range and panic instead of reading the talent's
// max-rank value. This is the mage's helper (sim/mage/talents.go),
// mirrored here: the mage half of the bug was fixed and the warrior
// half was not.
//
// Tables written one-based - the TalentSpellIDs arrays, which have no
// rank-0 entry - are read as rankIndex(rank-1, table), and every such
// call site returns early on rank 0, so the negative index never
// reaches here.
func rankIndex[T any](rank int32, table []T) int {
	if i := int(rank); i >= 0 && i < len(table) {
		return i
	}
	return len(table) - 1
}

// applyDeclarativeTalents is every talent that is a modifier on a set of
// spells. Before the spell-mod system these were OnSpellRegistered
// closures or arithmetic inlined into the ability file; as config they
// can be checked against a tooltip line by line, which is what Forever's
// weekly number changes need.
//
// Each ability's own file therefore carries its UNMODIFIED cost: the
// talent is applied here, once, and applying it in both places would
// double it.
func (warrior *Warrior) applyDeclarativeTalents() {
	t := warrior.Talents

	// Improved Heroic Strike: "Reduces the cost of your Heroic Strike
	// ability by 3 Rage" at rank 3, so 1 a point.
	if t.ImprovedHeroicStrike > 0 {
		warrior.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Flat,
			ClassMask: WarriorSpellMaskHeroicStrike,
			IntValue:  -int64(t.ImprovedHeroicStrike),
		})
	}

	// Improved Cleave: "Reduces the Rage cost of your Cleave ability" by
	// 1 a point. This is a CHANGED TALENT, not a renamed one: vanilla's
	// Improved Cleave raised Cleave's flat damage by 40% a point, and
	// the old body multiplied flatDamageBonus by {1, 1.4, 1.8, 2.2}.
	// The client's text gives a rage discount and no damage at all.
	if t.ImprovedCleave > 0 {
		warrior.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Flat,
			ClassMask: WarriorSpellMaskCleave,
			IntValue:  -int64(t.ImprovedCleave),
		})
	}

	// Improved Execute: the ranks are improvedExecuteRageReduction.
	if t.ImprovedExecute > 0 {
		warrior.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Flat,
			ClassMask: WarriorSpellMaskExecute,
			IntValue:  -improvedExecuteRageReduction[rankIndex(t.ImprovedExecute, improvedExecuteRageReduction[:])],
		})
	}

	// Improved Thunder Clap: "Reduces the Rage cost of your Thunder Clap
	// ability by 6" at rank 3, so 2 a point.
	if t.ImprovedThunderClap > 0 {
		warrior.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Flat,
			ClassMask: WarriorSpellMaskThunderClap,
			IntValue:  -2 * int64(t.ImprovedThunderClap),
		})
	}

	// Improved Sunder Armor: "by 3" at rank 3, so 1 a point.
	if t.ImprovedSunderArmor > 0 {
		warrior.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Flat,
			ClassMask: WarriorSpellMaskSunderArmor,
			IntValue:  -int64(t.ImprovedSunderArmor),
		})
	}

	// Two-Handed Weapon Specialization: "+3% damage with two-handed
	// melee weapons" at rank 3. The hand-type check cannot be expressed
	// as config, so the talent is skipped rather than applied and
	// filtered. SpellMod_DamageDone_Flat is additive percent: 1 = +1%.
	if t.TwoHandedWeaponSpecialization > 0 && warrior.MainHand().HandType == proto.HandType_HandTypeTwoHand {
		warrior.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_DamageDone_Flat,
			ClassMask: WarriorSpellMaskSpecials | WarriorSpellMaskOnNextSwing,
			IntValue:  int64(t.TwoHandedWeaponSpecialization),
		})
	}

	// Improved Overpower is applied in overpower.go, where the crit
	// bonus is a field of the one spell it names, and Impale in each
	// ability's CritDamageBonus: the client's Impale reads "your
	// abilities", which is every warrior spell rather than a mask group,
	// and a mask group would silently be the narrower of the two.
}

// impale is Impale: "Increases the critical strike damage bonus of your
// abilities by 20%" at rank 2, so 10% a point. It stays a helper rather
// than becoming a SpellModConfig because the client says "your
// abilities" without qualification, and every warrior spell reads it.
func (warrior *Warrior) impale() float64 {
	return 0.1 * float64(warrior.Talents.Impale)
}

func (warrior *Warrior) applyAngerManagement() {
	if !warrior.Talents.AngerManagement {
		return
	}

	rageMetrics := warrior.NewRageMetrics(core.ActionID{SpellID: TalentSpellIDs["anger_management"][0]})

	warrior.RegisterResetEffect(func(sim *core.Simulation) {
		core.StartPeriodicAction(sim, core.PeriodicActionOptions{
			Period: time.Second * 3,
			OnAction: func(sim *core.Simulation) {
				warrior.AddRage(sim, 1, rageMetrics)
				warrior.LastAMTick = sim.CurrentTime
			},
		})
	})
}

// applyWeaponmaster is Weaponmaster, the Arms talent that replaced
// vanilla's three weapon specializations (Sword, Axe, Polearm) with one
// node whose effect depends on what is equipped:
//
//	Axe/Polearm: +1% critical strike a point.
//	Mace/Staff:  ignore 3% of the target's armour a point.
//	Sword:       1% a point chance of an extra attack.
//
// The armour half is a percentage of the target's own armour, which is
// why it writes PseudoStats.ArmorIgnorePercent rather than a flat
// ArmorPenetration stat.
func (warrior *Warrior) applyWeaponmaster() {
	points := warrior.Talents.Weaponmaster
	if points == 0 {
		return
	}

	// Axe/Polearm: the character panel shows main-hand critical strike
	// chance only, so a main-hand-only match adds the stat and takes it
	// back off off-hand swings.
	switch warrior.GetProcMaskForTypes(proto.WeaponType_WeaponTypeAxe, proto.WeaponType_WeaponTypePolearm) {
	case core.ProcMaskMelee:
		warrior.AddStat(stats.Crit, core.CritRatingPerCritChance*float64(points))
	case core.ProcMaskMeleeMH:
		warrior.AddStat(stats.Crit, core.CritRatingPerCritChance*float64(points))
		warrior.OnSpellRegistered(func(spell *core.Spell) {
			if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
				spell.BonusCritRating -= core.CritRatingPerCritChance * float64(points)
			}
		})
	case core.ProcMaskMeleeOH:
		warrior.OnSpellRegistered(func(spell *core.Spell) {
			if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
				spell.BonusCritRating += core.CritRatingPerCritChance * float64(points)
			}
		})
	}

	// Mace/Staff: armour ignore is per attacker, not per weapon, so it
	// is applied once if either hand qualifies.
	if warrior.GetProcMaskForTypes(proto.WeaponType_WeaponTypeMace, proto.WeaponType_WeaponTypeStaff) != core.ProcMaskUnknown {
		warrior.PseudoStats.ArmorIgnorePercent += 0.03 * float64(points)
	}

	// Sword: an extra main-hand attack. The internal cooldown is
	// vanilla's Sword Specialization value and is unconfirmed for
	// Forever, which publishes no number for it.
	if mask := warrior.GetProcMaskForTypes(proto.WeaponType_WeaponTypeSword); mask != core.ProcMaskUnknown {
		warrior.registerWeaponmasterExtraAttack(mask, 0.01*float64(points))
	}
}

func (warrior *Warrior) registerWeaponmasterExtraAttack(procMask core.ProcMask, procChance float64) {
	icd := core.Cooldown{
		Timer:    warrior.NewTimer(),
		Duration: time.Millisecond * 200, // unconfirmed
	}
	actionID := core.ActionID{SpellID: TalentSpellIDs["weaponmaster"][0]}

	warrior.RegisterAura(core.Aura{
		Label:    "Weaponmaster",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(procMask) || !icd.IsReady(sim) {
				return
			}
			if sim.RandomFloat("Weaponmaster") < procChance {
				icd.Use(sim)
				warrior.AutoAttacks.ExtraMHAttack(sim, 1, actionID, spell.ActionID)
			}
		},
	})
}

// applyUnbridledWrath is Unbridled Wrath: "a 60% chance to generate 1
// additional Rage when you deal melee damage with a weapon ... increased
// to 2 Rage for two-handed weapons" at rank 5. Vanilla's was 8% a point
// and never doubled; the client's is 12% a point and does.
func (warrior *Warrior) applyUnbridledWrath() {
	if warrior.Talents.UnbridledWrath == 0 {
		return
	}

	procChance := 0.12 * float64(warrior.Talents.UnbridledWrath)
	rage := 1.0
	if warrior.MainHand().HandType == proto.HandType_HandTypeTwoHand {
		rage = 2.0
	}

	rageMetrics := warrior.NewRageMetrics(core.ActionID{SpellID: TalentSpellIDs["unbridled_wrath"][0]})

	warrior.RegisterAura(core.Aura{
		Label:    "Unbridled Wrath",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() {
				return
			}

			if spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) && sim.RandomFloat("Unbridled Wrath") < procChance {
				warrior.AddRage(sim, rage, rageMetrics)
			}
		},
	})
}

// applyDualWieldSpecialization is Dual Wield Specialization: "+25%
// off-hand weapon damage, +100% off-hand Rage generation, and +10%
// chance to hit with off-hand attacks" at rank 5. The damage half is
// vanilla's; the hit half is new, and is one of the thirteen hit and
// crit changes research/08-stats.md 1.2 records.
//
// unconfirmed: the off-hand rage-generation clause is not modelled. The
// engine derives rage from damage dealt rather than from a per-hand
// generation rate, so there is no hook for it that would not also
// change main-hand rage.
func (warrior *Warrior) applyDualWieldSpecialization() {
	points := warrior.Talents.DualWieldSpecialization
	if points == 0 {
		return
	}

	multiplier := 1 + 0.05*float64(points)
	bonusHit := core.HitRatingPerHitChance * 2 * float64(points)
	warrior.OnSpellRegistered(func(spell *core.Spell) {
		if !spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
			return
		}
		spell.BonusHitRating += bonusHit
		if spell.BonusCoefficient > 0 {
			spell.DamageMultiplier *= multiplier
		}
	})
}

// applyEnrage is Enrage: "a $h% chance to deal 10% increased Physical
// damage for 12 sec after being the victim of any damaging attack" at
// rank 5, so 2% a point - not vanilla's 5%.
//
// unconfirmed: the client leaves the trigger chance as an unresolved
// `$h`, and widens the trigger from "critically hit" to "any damaging
// attack". Until a number exists, the trigger stays the one vanilla
// used - a melee critical strike taken, at 100% - because guessing a
// chance for a much wider trigger would move the aura's uptime by more
// than the talent is worth. The validation job's aura-uptime comparison
// is what settles it.
func (warrior *Warrior) applyEnrage() {
	if warrior.Talents.Enrage == 0 {
		return
	}

	damageBonus := 1 + 0.02*float64(warrior.Talents.Enrage)

	warrior.EnrageAura = warrior.GetOrRegisterAura(core.Aura{
		Label:     "Enrage",
		ActionID:  core.ActionID{SpellID: TalentSpellIDs["enrage"][rankIndex(warrior.Talents.Enrage-1, TalentSpellIDs["enrage"])]},
		Duration:  time.Second * 12,
		MaxStacks: 12,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warrior.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= damageBonus
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warrior.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] /= damageBonus
		},
	})

	warrior.EnrageAura.NewExclusiveEffect("Enrage", true, core.ExclusiveEffect{Priority: 2 * float64(warrior.Talents.Enrage)})

	warrior.RegisterAura(core.Aura{
		Label:    "Enrage Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !warrior.EnrageAura.IsActive() {
				return
			}

			if spell.ProcMask.Matches(core.ProcMaskMelee) {
				warrior.EnrageAura.RemoveStack(sim)
			}
		},
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.ProcMask.Matches(core.ProcMaskMelee) {
				return
			}

			if !result.Outcome.Matches(core.OutcomeCrit) {
				return
			}

			warrior.EnrageAura.Activate(sim)
			if warrior.EnrageAura.IsActive() {
				warrior.EnrageAura.SetStacks(sim, 12)
			}
		},
	})
}

func (warrior *Warrior) applyFlurry() {
	if warrior.Talents.Flurry == 0 {
		return
	}

	talentAura := warrior.makeFlurryAura(warrior.Talents.Flurry)

	// This must be registered before the below trigger because in-game a crit weapon swing consumes a stack before the refresh, so you end up with:
	// 3 => 2
	// refresh
	// 2 => 3
	warrior.makeFlurryConsumptionTrigger(talentAura)

	core.MakePermanent(warrior.RegisterAura(core.Aura{
		Label: "Flurry Proc Trigger",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.ProcMask.Matches(core.ProcMaskMelee) && result.Outcome.Matches(core.OutcomeCrit) {
				talentAura.Activate(sim)
				if talentAura.IsActive() {
					talentAura.SetStacks(sim, 3)
				}
				return
			}
		},
	}))
}

// makeFlurryAura is separated out because of the T1 Shaman Tank 2P that
// can proc Flurry separately from the talent. It triggers the max-rank
// Flurry aura but with dodge, parry, or block.
//
// The client's Flurry is "+25% melee attack speed for your next swings"
// at rank 5, so 5% a point. Vanilla's was 10% at rank 1 rising to 30%;
// this is the single largest number change in the Fury tree and the
// reason the spec's goldens move.
func (warrior *Warrior) makeFlurryAura(points int32) *core.Aura {
	if points == 0 {
		return nil
	}

	// The client gives every rank of Flurry the same rank spell, 12319,
	// so the id no longer distinguishes one rank's aura from another's
	// and the label carries the rank instead. It has to: the
	// Protection T2 4pc registers its own five-point Flurry through
	// this same function, and GetOrRegisterAura keys on the label, so
	// two ranks sharing a label would silently become one aura at
	// whichever attack speed registered first.
	spellID := TalentSpellIDs["flurry"][rankIndex(points-1, TalentSpellIDs["flurry"])]
	attackSpeed := flurryAttackSpeed[rankIndex(points, flurryAttackSpeed[:])]

	aura := warrior.GetOrRegisterAura(core.Aura{
		Label:     fmt.Sprintf("Flurry Proc (%d points)", points),
		ActionID:  core.ActionID{SpellID: spellID},
		Duration:  core.NeverExpires,
		MaxStacks: 3,
	})

	aura.NewExclusiveEffect("Flurry", true, core.ExclusiveEffect{
		Priority: attackSpeed,
		OnGain: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			warrior.MultiplyMeleeSpeed(sim, attackSpeed)
		},
		OnExpire: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			warrior.MultiplyMeleeSpeed(sim, 1/attackSpeed)
		},
	})

	return aura
}

// With the Protection T2 4pc it's possible to have 2 different Flurry auras if using less than 5/5 points in Flurry.
// The two different buffs don't stack whatsoever. Instead the stronger aura takes precedence and each one is only refreshed by the corresponding triggers.
func (warrior *Warrior) makeFlurryConsumptionTrigger(flurryAura *core.Aura) *core.Aura {
	icd := core.Cooldown{
		Timer:    warrior.NewTimer(),
		Duration: time.Millisecond * 500,
	}
	return core.MakePermanent(warrior.GetOrRegisterAura(core.Aura{
		Label: "Flurry Consume Trigger - " + flurryAura.Label,
		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			// Remove a stack.
			if flurryAura.IsActive() && spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) && icd.IsReady(sim) {
				icd.Use(sim)
				flurryAura.RemoveStack(sim)
			}
		},
	}))
}

// applyShieldSpecialization is Shield Specialization: "+5% chance to
// Block and a 100% chance to generate 5 Rage when you Block" at rank 5,
// so 1% block and a 20%-a-point chance of 5 rage.
func (warrior *Warrior) applyShieldSpecialization() {
	if warrior.Talents.ShieldSpecialization == 0 {
		return
	}

	warrior.AddStat(stats.Block, core.BlockRatingPerBlockChance*1*float64(warrior.Talents.ShieldSpecialization))

	procChance := 0.2 * float64(warrior.Talents.ShieldSpecialization)
	rageMetrics := warrior.NewRageMetrics(core.ActionID{SpellID: TalentSpellIDs["shield_specialization"][rankIndex(warrior.Talents.ShieldSpecialization-1, TalentSpellIDs["shield_specialization"])]})

	warrior.RegisterAura(core.Aura{
		Label:    "Shield Specialization",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidBlock() {
				if sim.Proc(procChance, "Shield Specialization") {
					warrior.AddRage(sim, 5.0, rageMetrics)
				}
			}
		},
	})
}

// registerDeathWishCD is Death Wish: "increases your Physical damage
// done by 20% and makes you immune to Fear effects, but increases all
// damage you take by 5%. Lasts 30 sec." The old body modelled the
// downside as a 20% armour loss, which is neither what the client says
// nor equivalent to it against magic damage.
func (warrior *Warrior) registerDeathWishCD() {
	if !warrior.Talents.DeathWish {
		return
	}

	actionID := core.ActionID{SpellID: TalentSpellIDs["death_wish"][0]}

	deathWishAura := warrior.RegisterAura(core.Aura{
		Label:    "Death Wish",
		ActionID: actionID,
		Duration: time.Second * 30,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warrior.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= 1.2
			warrior.PseudoStats.DamageTakenMultiplier *= 1.05
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warrior.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] /= 1.2
			warrior.PseudoStats.DamageTakenMultiplier /= 1.05
		},
	})
	core.RegisterPercentDamageModifierEffect(deathWishAura, 1.2)

	warrior.DeathWish = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagHelpful | core.SpellFlagAPL,
		RageCost: core.RageCostOptions{
			Cost: 10,
		},
		Cast: core.CastConfig{
			IgnoreHaste: true,
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: time.Minute * 3,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			deathWishAura.Activate(sim)
		},
	})

	warrior.AddMajorCooldown(core.MajorCooldown{
		Spell: warrior.DeathWish.Spell,
		Type:  core.CooldownTypeDPS,
	})
}

func (warrior *Warrior) registerLastStandCD() {
	if !warrior.Talents.LastStand {
		return
	}

	actionID := core.ActionID{SpellID: TalentSpellIDs["last_stand"][0]}
	healthMetrics := warrior.NewHealthMetrics(actionID)

	var bonusHealth float64
	lastStandAura := warrior.RegisterAura(core.Aura{
		Label:    "Last Stand",
		ActionID: actionID,
		Duration: time.Second * 20,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			bonusHealth = warrior.MaxHealth() * 0.3
			warrior.AddStatsDynamic(sim, stats.Stats{stats.Health: bonusHealth})
			warrior.GainHealth(sim, bonusHealth, healthMetrics)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warrior.AddStatsDynamic(sim, stats.Stats{stats.Health: -bonusHealth})
		},
	})

	lastStandSpell := warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID: actionID,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: time.Minute * 10,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			lastStandAura.Activate(sim)
		},
	})

	warrior.AddMajorCooldown(core.MajorCooldown{
		Spell: lastStandSpell.Spell,
		Type:  core.CooldownTypeSurvival,
	})
}

// Not modelled, and deliberately so rather than by omission. Each is in
// the client's tree and each would need machinery this spec does not
// have.
//
// Three of them ARE in a reference build, so the cost is stated rather
// than denied. ForeverFuryTalents spends 1 point on Booming Voice - the
// cheapest legal filler on the Fury tier-0 row - and that is the one
// point of the Fury build's 51 that buys nothing at all.
// ForeverProtectionTalents spends 1 on Concussion Blow and 2 on
// Bastion, which cost nothing today only because sim/warrior/tank_warrior
// is skipped; the warrior-protection spec's task inherits them.
//
// Improved Tactical Mastery used to belong on this list and no longer
// does: its rank text is a plain retained-rage number and stances.go
// models it.
//
//	Booming Voice        - "+50% Battle Shout and Demoralizing Shout
//	                       radius" at rank 5. Vanilla's raised their
//	                       duration, which core.BattleShoutAura still
//	                       takes a points argument for; radius has no
//	                       meaning in a raid sim, so shouts.go passes 0.
//	Iron Will            - stun and fear duration; no encounter models one.
//	Blood Craze          - a self heal-over-time.
//	Boundless Rage       - "+30 maximum Rage" at rank 3. core.MaxRage is
//	                       a package constant, so this needs a core
//	                       change, which is another task's file.
//	Raging Blows         - Whirlwind also strikes off-hand.
//	Bloodthrill          - an Overpower activation proc off Rend.
//	Improved Intercept,
//	Improved Charge,
//	Improved Hamstring,
//	Improved Disarm,
//	Vanguard,
//	Improved Shield Bash - abilities or situations no Patchwerk profile uses.
//	Spearing Strike,
//	Concussion Blow,
//	Master of Defense,
//	Focused Rage,
//	Bastion,
//	Weaponmaster's
//	 dismount clause      - Protection and utility, for the warrior-protection spec.
