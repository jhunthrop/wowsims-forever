package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Whirlwind is the one ability in this package that reads NOTHING out
// of constants_auto_gen.go, and the third row the data lane owes a fix
// for. Column by column:
//
//   - WhirlwindManaCost[0] and WhirlwindCooldownMS[0] are both 0. Two
//     rows share the name, the rank (0) and spell_level 36 - the
//     250-tenths, 10 s 1680 and a free reissue, 462891 - and the dedup
//     kept the free one. The two constants below are therefore read by
//     hand from data/builds/1.60.1.69893/spellconst/warrior.json, which
//     gives 1680 cost 250 tenths and category_cooldown_ms 10000.
//   - WhirlwindSpellId[0] is 462891 for the same reason, where the UI
//     and the pinned rotation both name 1680. Ids are not swapped here.
//   - WhirlwindBaseDamage is {0, 0}, and that IS sound: the client says
//     Whirlwind is pure weapon damage with no flat term. There is
//     simply nothing to read, which is why ApplyEffects adds no base
//     damage rather than adding a generated zero.
const (
	whirlwindRageCost = 25.0
	whirlwindCooldown = time.Second * 10
)

// Improved Whirlwind (Fury node 105977, 2 ranks, spell 1310315; the node
// held Raging Blows until 1.60.1.70755): "Reduces the cooldown of your
// Whirlwind ability by 1/2 sec and increases the damage it deals to the
// closest enemy by 15%/30%." The rows: e0 aura 107 (add flat modifier)
// misc 11 (cooldown) on Whirlwind's family bit (mask_1 0x4), trait
// curve 128139 (1: -1000, 2: -2000 ms); e1 aura 4 (dummy, a server
// script) with trait curve 128138 (1: 15, 2: 30). The engine's first
// target is the closest one: the +% lands on the main- and off-hand
// strikes Whirlwind deals to Whirlwind's primary target and to no other.
var (
	improvedWhirlwindCooldownCutMS         = [3]int32{0, 1000, 2000}
	improvedWhirlwindPrimaryTargetBonusPct = [3]float64{0, 15, 30}
)

// improvedWhirlwindCooldownCut is what Improved Whirlwind takes off
// Whirlwind's cooldown at a point count.
func improvedWhirlwindCooldownCut(points int32) time.Duration {
	return time.Duration(improvedWhirlwindCooldownCutMS[rankIndex(points, improvedWhirlwindCooldownCutMS[:])]) * time.Millisecond
}

// improvedWhirlwindPrimaryTargetMultiplier is the damage multiplier
// Improved Whirlwind gives Whirlwind's strikes on its closest target.
func improvedWhirlwindPrimaryTargetMultiplier(points int32) float64 {
	return 1 + improvedWhirlwindPrimaryTargetBonusPct[rankIndex(points, improvedWhirlwindPrimaryTargetBonusPct[:])]/100
}

// WhirlwindOffHandSpellID is Whirlwind's off-hand strike, the client's
// spell 462891: SpellMisc Attributes_3 0x01000000 (the off-hand attack
// flag), effect 121 normalized weapon damage on one target, no cost and no
// stance, under Whirlwind's family mask.
const WhirlwindOffHandSpellID = 462891

// knowsWhirlwind reports whether the warrior has learned Whirlwind
// (SpellLevels BaseLevel 36 for 1680).
func (warrior *Warrior) knowsWhirlwind() bool {
	return int(warrior.Level) >= WhirlwindLevel[0]
}

func (warrior *Warrior) registerWhirlwindSpell() {
	// Sized from the target POOL, not the active prefix: a timeline can
	// start small and grow, and a slice fixed at the starting count
	// would cap the ability there for the rest of the fight. The cast
	// loop below bounds itself by however many targets are active at
	// that moment, so a shrinking timeline does not wrap surplus swings
	// back onto the survivors. Same shape as APLActionMultidot.
	results := make([]*core.SpellResult, min(4, len(warrior.Env.Encounter.AllTargetUnits)))
	// ohResults backs the off-hand strike (below); sized the same as
	// results so the two loops share one numHits bound.
	ohResults := make([]*core.SpellResult, len(results))
	offHandStrike := warrior.registerWhirlwindOffHandStrike()
	primaryTargetMultiplier := improvedWhirlwindPrimaryTargetMultiplier(warrior.Talents.ImprovedWhirlwind)

	warrior.Whirlwind = warrior.RegisterSpell(BerserkerStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorWhirlwind,
		ClassSpellMask: WarriorSpellMaskWhirlwind,
		ActionID:       core.ActionID{SpellID: 1680},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: WhirlwindLevel[0],
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.knowsWhirlwind()
		},

		RageCost: core.RageCostOptions{
			Cost: whirlwindRageCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: whirlwindCooldown - improvedWhirlwindCooldownCut(warrior.Talents.ImprovedWhirlwind),
			},
		},
		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1.25,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Whirlwind strikes with both weapons baseline (Blizzard's
			// 1 October 2026 notes).
			// A dual-wielding check because the clause presumes a weapon
			// is there to strike with; a 2H Whirlwind swings with the
			// main hand only, same as always.
			offHand := warrior.AutoAttacks.IsDualWielding

			numHits := min(len(results), len(sim.Encounter.TargetUnits))
			for idx := 0; idx < numHits; idx++ {
				// Improved Whirlwind's bonus is the closest enemy's
				// alone: the cast's own target, the first one struck.
				targetMultiplier := 1.0
				if idx == 0 {
					targetMultiplier = primaryTargetMultiplier
				}
				baseDamage := targetMultiplier * spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
				results[idx] = spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
				if offHand {
					ohDamage := targetMultiplier * offHandStrike.Unit.OHNormalizedWeaponDamage(sim, offHandStrike.MeleeAttackPower(target))
					ohResults[idx] = offHandStrike.CalcDamage(sim, target, ohDamage, offHandStrike.OutcomeMeleeWeaponSpecialHitAndCrit)
				}
				target = sim.Environment.NextTargetUnit(target)
			}

			for _, result := range results[:numHits] {
				spell.DealDamage(sim, result)
			}
			if offHand {
				for _, result := range ohResults[:numHits] {
					offHandStrike.DealDamage(sim, result)
				}
			}
		},
	})
}

// registerWhirlwindOffHandStrike registers Whirlwind's off-hand strike
// (WhirlwindOffHandSpellID) as the off-hand special the client makes it, so
// it rolls the off hand's procs and the off hand's talents (Dual Wield
// Specialization's damage, Furious Precision's hit) through their own
// off-hand masks, and never the main hand's (Windfury Totem, a main-hand
// enchant). Whirlwind casts it; it has no cost, cooldown or stance of its
// own.
func (warrior *Warrior) registerWhirlwindOffHandStrike() *core.Spell {
	return warrior.Unit.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_WarriorWhirlwind,
		ClassSpellMask: WarriorSpellMaskWhirlwind,
		ActionID:       core.ActionID{SpellID: WhirlwindOffHandSpellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeOHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell | core.SpellFlagNoOnCastComplete,

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1.25,
		BonusCoefficient: 1,
	})
}
