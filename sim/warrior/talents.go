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
// It is written against the live tree (the client's trait tables with
// Wowhead's hotfix overlay, build 1.60.1.70009), so its three segments
// are 17, 17 and 18 characters (TalentTreeSizes). Spend:
//
//	Arms 20: Improved Heroic Strike 3, Deflection 3, Improved Rend 3,
//	         Improved Tactical Mastery 5, Anger Management 1,
//	         Deep Wounds 3, Impale 2
//	Fury 31: Booming Voice 1, Cruelty 5, Lingering Rage 3,
//	         Unbridled Wrath 5, Furious Precision 3, Piercing Howl 1,
//	         Enrage 5, Improved Execute 1, Death Wish 1, Flurry 5,
//	         Bloodthirst 1
//	Protection 0
//
// The live-tree rebuild removed Improved Cleave 3 and Precision 3 from
// the earlier reference build; their six points went to Furious
// Precision 3 (the replacement hit talent) and Lingering Rage 3, which
// has no effect on a simmed number, so the reference build's DPS moves
// only by what the removed and added talents themselves are worth.
//
// Improved Rend is 3, not the 2 an earlier draft spent, because the
// client makes Deep Wounds require Improved Rend at rank 3; and the Fury
// half is shaped by the tier gates rather than by taste, because
// reaching Bloodthirst with exactly 31 points forces 25 of them into
// tiers 0-4. TestForeverFuryTalentsAreAValidBuild checks the widths and
// the spend, so a segment one character short fails rather than
// silently reading the neighbouring talent.
const ForeverFuryTalents = "33305013002000000-15353100051010501-000000000000000000"

// ForeverProtectionTalents is the same fixed input for the tank spec: a
// 20-point Arms spend that a tank wants (the parry and the cheaper Heroic
// Strike) and a 31-point Protection spend that reaches Shield Slam, the
// 31-point talent. It is not advice, it is a fixed input, so a tank number
// that moves is attributable to the engine and not to a build edit.
//
//	Arms 20:       Improved Heroic Strike 3, Deflection 5,
//	               Improved Rend 3, Improved Tactical Mastery 5,
//	               Anger Management 1, Deep Wounds 3
//	Protection 31: Shield Specialization 5, Anticipation 5,
//	               Improved Revenge 3, Improved Thunder Clap 3,
//	               Last Stand 1, Master of Defense 2, Defiance 3,
//	               Improved Sunder Armor 3, Concussion Blow 1,
//	               Focused Rage 3, Bastion 1, Shield Slam 1
//
// The Protection spend is shaped by the tier gates: Shield Slam (tier 6)
// needs 30 points above it, Concussion Blow (tier 4) is its prerequisite,
// and Master of Defense needs Shield Specialization at 5. Iron Will,
// Vanguard and the two Improved Shield/Disarm talents are left out: a
// stationary one-boss tank sim stuns, silences and disarms nothing.
// TestTheReferenceBuildsRespectTheTiersAndPrerequisites checks the widths
// and the gates.
const ForeverProtectionTalents = "35305013000000000-00000000000000000-050533120330001311"

// fillWarriorTalents parses a talent string into the proto, positionally
// against TalentTreeSizes. It is the one place that pairing happens, so
// a tree-size change cannot be applied in one caller and missed in
// another.
func fillWarriorTalents(talents *proto.WarriorTalents, s string) {
	core.FillTalentsProto(talents.ProtoReflect(), s, TalentTreeSizes)
}

func (warrior *Warrior) ApplyTalents() {
	// Flat stats. Forever has no combat ratings, so a percentage is the
	// stat: CritRatingPerCritChance and HitRatingPerHitChance are both 1.
	//
	//	Cruelty:      "+5% melee critical strike" at rank 5.
	//	Anticipation: "+20 Defense Skill" at rank 5, so 4 a point - twice
	//	              vanilla's 2, which is what the old body applied.
	//	Deflection:   "+5% Parry" at rank 5.
	warrior.AddStat(stats.Crit, core.CritRatingPerCritChance*1*float64(warrior.Talents.Cruelty))
	warrior.AddStat(stats.Defense, 4*float64(warrior.Talents.Anticipation))
	warrior.AddStat(stats.Parry, 1*float64(warrior.Talents.Deflection))

	warrior.applyDeclarativeTalents()

	// Talents with real mechanics keep their own functions.
	warrior.applyAngerManagement()
	warrior.applyDeepWounds()
	warrior.applyWeaponmaster()
	warrior.applyUnbridledWrath()
	warrior.applyDualWieldSpecialization()
	warrior.applyFuriousPrecision()
	warrior.applyEnrage()
	warrior.applyFlurry()
	warrior.applyProtectionTalents()
	warrior.applyImprovedCharge()
	warrior.applyBloodthrill()
	warrior.applyBloodCraze()
	warrior.registerDeathWishCD()
	warrior.registerSweepingStrikesCD()
}

// applyImprovedCharge is Improved Charge: "Increases the Rage generated
// by your Charge ability by 3/6" at ranks 1-2 (Arms node 105955).
//
// This package registers no Charge spell at all - neither this fork nor
// upstream models the gap-closer, which would need a movement primitive
// core does not have (PORTING.md's Encounter.movement is the target
// moving, not the player) - so the rage is granted once per sim, at
// Reset, standing in for the opening Charge every melee profile uses to
// start the pull. It is granted unconditionally rather than gated on the
// fight's starting stance: the charge-in happens before combat and
// before Reset's stance switch runs, so by the time a stance check could
// read Vanguard or the input stance, the rage would already be in the
// bar. Improved Intercept, the Fury-side sibling, grants no rage at all
// (just a cooldown reduction) and is in the not-modelled list below.
var improvedChargeRagePerRank = [3]float64{0, 3, 6}

func (warrior *Warrior) applyImprovedCharge() {
	if warrior.Talents.ImprovedCharge == 0 {
		return
	}

	rage := improvedChargeRagePerRank[rankIndex(warrior.Talents.ImprovedCharge, improvedChargeRagePerRank[:])]
	rageMetrics := warrior.NewRageMetrics(core.ActionID{SpellID: TalentSpellIDs["improved_charge"][0]})

	warrior.RegisterResetEffect(func(sim *core.Simulation) {
		// Deferred one event past Reset itself: Unit.reset runs the
		// registered reset effects before unit.rageBar.reset (sim/core/
		// unit.go), so an AddRage called directly here would be
		// overwritten the moment the rage bar resets to its own
		// starting value a few lines later in the same Reset pass.
		// DoAt at the current (reset) time queues this for the start of
		// the event loop instead, strictly after every Reset call has
		// run.
		core.StartDelayedAction(sim, core.DelayedActionOptions{
			DoAt: sim.CurrentTime,
			OnAction: func(sim *core.Simulation) {
				warrior.AddRage(sim, rage, rageMetrics)
			},
		})
	})
}

// Talents whose ranks do not scale linearly, or whose values are the
// client's own per-rank figures, need a table rather than a
// multiplication. They live here, in one block, so rankIndex below is
// the only way any of them is read; each is named at the site that
// reads it, which may be another file in this package.
var (
	// Improved Execute: "by 3" at rank 1 and "by 5" at rank 2 - not
	// vanilla's 2 and 5, which is what the old inline table said.
	improvedExecuteRageReduction = [3]int64{0, 3, 5}
	// Improved Rend: "Increases the damage of your Rend ability by
	// 35%" at rank 3.
	improvedRendDamageMultiplier = [4]float64{1, 1.12, 1.23, 1.35}
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

	// Improved Cleave, Precision, Toughness and Boundless Rage are gone
	// from the live tree (the 1 October 2026 Fury rebuild and the
	// Protection rework); their fields left the generated proto.

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
	// filtered.
	//
	// It is a school multiplier, not a spell mod: the tooltip says
	// "damage with two-handed melee weapons", and for a 2H warrior the
	// largest share of that is auto-attacks, which carry no
	// ClassSpellMask at all - a mask-based mod reaches neither them nor
	// Rend, Thunder Clap, Revenge or Sunder Armor, which is every
	// warrior ability outside the two mask groups it named. The
	// hand-type guard is what keeps it from also covering the spells a
	// dual-wielding or sword-and-board warrior casts, and
	// SchoolDamageDealtMultiplier[Physical] is the pattern Death Wish
	// (applyDeathWish) and Enrage (applyEnrage) already use.
	if t.TwoHandedWeaponSpecialization > 0 && warrior.MainHand().HandType == proto.HandType_HandTypeTwoHand {
		warrior.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= 1 + 0.01*float64(t.TwoHandedWeaponSpecialization)
	}

	// Raging Blows: the client text reads "reduces the Rage cost of your
	// Cleave ability by 2"; Blizzard's 1 October 2026 notes say it now
	// "reduces the rage cost of Cleave and Whirlwind by 3", and the note
	// is the live state (hotfix), so both abilities lose 3. The
	// off-hand Whirlwind strike the client text also names is baseline
	// since the same notes (whirlwind.go).
	if t.RagingBlows {
		warrior.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Flat,
			ClassMask: WarriorSpellMaskCleave | WarriorSpellMaskWhirlwind,
			IntValue:  -ragingBlowsRageDiscount,
		})
	}

	// Improved Revenge: "Increases damage dealt by your Revenge ability
	// by 20%/40%/60%" at ranks 1-3 (Protection node 105969, tank-only): a
	// flat percentage on one named spell, the same shape as every other
	// declarative mod here.
	if t.ImprovedRevenge > 0 {
		warrior.AddStaticMod(core.SpellModConfig{
			Kind:      core.SpellMod_DamageDone_Flat,
			ClassMask: WarriorSpellMaskRevenge,
			IntValue:  20 * int64(t.ImprovedRevenge),
		})
	}

	// Bastion: "Increases all damage you deal by 2%/4%/6%/8%/10% while a
	// shield is equipped" at ranks 1-5 (Protection node 105962,
	// tank-only). Like Two-Handed Weapon Specialization above, "all
	// damage" is wider than any ClassMask or ClassSpellsOnly reaches -
	// neither covers the auto-attack that is most of a shield warrior's
	// damage - and the shield check is gear state a SpellModConfig
	// cannot express, so this goes directly on PseudoStats instead of
	// becoming a SpellMod. warrior.PseudoStats.CanBlock is already
	// exactly this condition (character.go sets it from
	// OffHand().WeaponType == WeaponTypeShield), reused rather than
	// re-read.
	if t.Bastion > 0 && warrior.PseudoStats.CanBlock {
		warrior.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= 1 + 0.02*float64(t.Bastion)
	}

	// Focused Rage: "Reduces the Rage cost of your offensive abilities
	// by 1/2/3" at ranks 1-3 (Protection node 105961, tank-only).
	// SpellFlagOffensive (warrior.go) is carried by every rage-costing
	// special in this package - exactly "your offensive abilities" - so
	// the SpellFlags filter reaches the whole set in one mod instead of
	// a ClassMask union of every offensive spell's own mask.
	if t.FocusedRage > 0 {
		warrior.AddStaticMod(core.SpellModConfig{
			Kind:       core.SpellMod_PowerCost_Flat,
			SpellFlags: SpellFlagOffensive,
			IntValue:   -int64(t.FocusedRage),
		})
	}

	// Improved Overpower is applied in overpower.go, where the crit
	// bonus is a field of the one spell it names, and Impale in each
	// ability's CritDamageBonus: the client's Impale reads "your
	// abilities", which is every warrior spell rather than a mask group,
	// and a mask group would silently be the narrower of the two.

	// Improved Intercept: "Reduces the cooldown of your Intercept
	// ability by 5/10 sec." This package registers no Intercept spell -
	// see applyImprovedCharge's comment on why Charge and Intercept are
	// both unregistered - and even with one, a stationary Patchwerk
	// target gives no reason to re-engage from range a second time, so a
	// cooldown on an ability the rotation never recasts changes nothing.
	_ = t.ImprovedIntercept

	// Improved Hamstring: "Gives your Hamstring ability a 5/10/15%
	// chance to immobilize the target for 5 sec." A root changes nothing
	// on this package's stationary-target encounters; nothing here
	// simulates the movement a root would prevent.
	_ = t.ImprovedHamstring

	// Lingering Rage (Fury node 110857, hotfix_only): "Increases the time
	// before your Rage begins to decay after leaving combat by 2/4/6/8/10
	// sec." Rage decays only out of combat, and a DPS encounter never
	// leaves combat, so it has no effect on a simmed number.
	_ = t.LingeringRage

	// Gore Drinker (Fury node 113569, hotfix_only, needs Enrage x5):
	// "Your Enrage, Berserker Rage, Bloodrage, Death Wish, and
	// Bloodthirst abilities cause your next 3 melee attacks to restore
	// 1.0% of your maximum Health." Healing the warrior changes no damage
	// number and the sim has no warrior health model, so it is an
	// explicit no-DPS-effect talent.
	_ = t.GoreDrinker

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

// unbridledWrathRage is the rage one proc generates. The pre-1 October
// client text doubled it to 2 for two-handed weapons; the live text and
// Blizzard's 1 October 2026 notes ("Unbridled Wrath 1 rage regardless of
// weapon") give a flat 1.
const unbridledWrathRage = 1.0

// applyUnbridledWrath is Unbridled Wrath: "a 60% chance to generate 1
// additional Rage when you deal melee damage with a weapon" at rank 5,
// 12% a point, regardless of weapon (see unbridledWrathRage).
func (warrior *Warrior) applyUnbridledWrath() {
	if warrior.Talents.UnbridledWrath == 0 {
		return
	}

	procChance := 0.12 * float64(warrior.Talents.UnbridledWrath)
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
				warrior.AddRage(sim, unbridledWrathRage, rageMetrics)
			}
		},
	})
}

// dualWieldSpecializationOffHandRageMultiplier is the talent's rage
// clause, 10% a point ("the Rage generated by your off-hand attacks by
// 50%" at rank 5; Blizzard's 1 October 2026 notes: "off-hand rage
// 10/20/30/40/50%").
func dualWieldSpecializationOffHandRageMultiplier(points int32) float64 {
	return 1 + 0.1*float64(points)
}

// furiousPrecisionOffHandHitPercent is Furious Precision's off-hand hit,
// per the rank text "by 4%", "7%", "10%".
func furiousPrecisionOffHandHitPercent(points int32) float64 {
	return [4]float64{0, 4, 7, 10}[max(0, min(points, 3))]
}

// ragingBlowsRageDiscount is the rage Raging Blows takes off Cleave and
// Whirlwind: 3 per Blizzard's 1 October 2026 notes (client text: 2, Cleave
// only).
const ragingBlowsRageDiscount = 3

// applyFuriousPrecision is Furious Precision (Fury node 105953,
// hotfix_only): "Increases your chance to hit with off-hand attacks by
// 4/7/10%."
func (warrior *Warrior) applyFuriousPrecision() {
	bonusHit := core.HitRatingPerHitChance * furiousPrecisionOffHandHitPercent(warrior.Talents.FuriousPrecision)
	if bonusHit == 0 {
		return
	}
	warrior.OnSpellRegistered(func(spell *core.Spell) {
		if spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
			spell.BonusHitRating += bonusHit
		}
	})
}

// applyDualWieldSpecialization is Dual Wield Specialization: "+25%
// off-hand weapon damage and +50% off-hand Rage generated" at rank 5.
// The hit clause the pre-1 October text carried is gone (it is Furious
// Precision now; Blizzard's notes: "no hit").
func (warrior *Warrior) applyDualWieldSpecialization() {
	points := warrior.Talents.DualWieldSpecialization
	if points == 0 {
		return
	}

	multiplier := 1 + 0.05*float64(points)
	warrior.AddOffHandDamageDealtRageMultiplier(dualWieldSpecializationOffHandRageMultiplier(points))
	warrior.OnSpellRegistered(func(spell *core.Spell) {
		if !spell.ProcMask.Matches(core.ProcMaskMeleeOH) {
			return
		}
		// The damage half is weapon damage, so it is narrowed by school
		// rather than by BonusCoefficient, which is what this line used
		// to read. The two select the same spells today - an
		// auto-attack's coefficient is 1 exactly when its school is
		// Physical (core/attack.go:494), and the off-hand auto is the
		// only ProcMaskMeleeOH spell a warrior registers - but the
		// coefficient is a proxy for the school and reads as though a
		// physical swing were excluded from its own talent.
		if spell.SpellSchool.Matches(core.SpellSchoolPhysical) {
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

		RequiredLevel: DeathWishLevel[0],

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

// Not modelled, and deliberately so rather than by omission. Each is in
// the client's tree and each would need machinery this spec does not
// have. Improved Intercept, Improved Hamstring and Lingering Rage are named
// at the `_ = t.X` lines at the end of applyDeclarativeTalents instead,
// with their reasons beside them, which is where a reader already is when a
// mask or a mod is missing for one of them; the Protection tree's
// (Iron Will, Improved Disarm, Vanguard, Improved Shield Bash and Concussion
// Blow) are listed at the end of talents_protection.go.
//
// ForeverFuryTalents spends 1 point on Booming Voice - the cheapest
// legal filler on the Fury tier-0 row - and that is the one point of the
// Fury build's 51 that buys nothing at all.
//
// Improved Tactical Mastery used to belong on this list and no longer
// does: its rank text is a plain retained-rage number and stances.go
// models it.
//
//	Booming Voice     - "+50% Battle Shout and Demoralizing Shout
//	                    radius" at rank 5. Vanilla's raised their
//	                    duration, which core.BattleShoutAura still takes
//	                    a points argument for; radius has no meaning in
//	                    a raid sim, so shouts.go passes 0.
//	Weaponmaster's
//	 dismount clause - Protection and utility, for the warrior-protection spec.
