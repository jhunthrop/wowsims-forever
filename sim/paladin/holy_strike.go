package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// HolyStrikeFlatDamage is effect 121's flat bonus for ids 679 through
// 10333, spellconst/paladin.json's own roll: rank 8 is 93 at level 60,
// 0.25 wide (81.4-104.6 at the rank's own level) and growing 3.2 a level
// to level 65, where the flat bonuses it replaced were one number per
// rank with no width and no growth.
var HolyStrikeFlatDamage = [holyStrikeRankCount + 1]clientdamage.Effect{
	{},
	{Amount: 12, Variance: 0.25, PerLevel: 0.1, SpellLevel: 6, MaxLevel: 11},
	{Amount: 17, Variance: 0.25, PerLevel: 0.1, SpellLevel: 12, MaxLevel: 17},
	{Amount: 19, Variance: 0.25, PerLevel: 0.2, SpellLevel: 20, MaxLevel: 25},
	{Amount: 24, Variance: 0.25, PerLevel: 0.3, SpellLevel: 28, MaxLevel: 33},
	{Amount: 31, Variance: 0.25, PerLevel: 1, SpellLevel: 36, MaxLevel: 41},
	{Amount: 53, Variance: 0.25, PerLevel: 1.5, SpellLevel: 44, MaxLevel: 49},
	{Amount: 68, Variance: 0.25, PerLevel: 2.8, SpellLevel: 52, MaxLevel: 57},
	{Amount: 93, Variance: 0.25, PerLevel: 3.2, SpellLevel: 60, MaxLevel: 65},
}

const holyStrikeRankCount = 8

// holyStrikeRanks: source 1.60.1.70009 client spell data
// (spellconst/paladin.json, "Holy Strike"). Holy Strike is new in Forever:
// there was no engine file for it before this change. It is an instant
// Holy-school weapon attack (school_mask 2, cast_time_ms 0, gcd_ms 1500)
// on a flat 10 s cooldown (cooldown_ms 0, category_cooldown_ms 10000) for
// every rank.
//
// Each rank has three effects: index 0 (effect 121, normalized weapon
// damage) gives flatBonus, a flat bonus added on top of normalized main
// hand weapon damage, plus spell power scaling via sp_coefficient (a
// constant 0.429 across all eight ranks -- see BonusCoefficient below,
// same coefficient Exorcism uses); index 1 (effect 31, weapon percent
// damage) gives percentOfWeapon, applied the way this fork's other
// weapon-percent-damage attacks apply their percentage (e.g.
// sim/rogue/backstab.go's 150%): as the spell's DamageMultiplier over the
// same (flatBonus + normalized weapon damage) base, not as a second,
// additive weapon-damage instance; index 2 (effect 77, script effect) has
// no base_points at any rank (always 0) and appears to be a scripting
// hook Forever uses server-side (Sacred Arbiter's refresh-Judgement
// behavior is the likely candidate) rather than a value this engine needs
// to read.
var holyStrikeRanks = []struct {
	level           int32
	spellID         int32
	manaCost        float64
	percentOfWeapon float64
}{
	{level: 6, spellID: 679, manaCost: 5, percentOfWeapon: 25},
	{level: 12, spellID: 678, manaCost: 9, percentOfWeapon: 29},
	{level: 20, spellID: 1866, manaCost: 12, percentOfWeapon: 32},
	{level: 28, spellID: 680, manaCost: 14, percentOfWeapon: 36},
	{level: 36, spellID: 2495, manaCost: 16, percentOfWeapon: 39},
	{level: 44, spellID: 5569, manaCost: 17, percentOfWeapon: 43},
	{level: 52, spellID: 10332, manaCost: 19, percentOfWeapon: 46},
	{level: 60, spellID: 10333, manaCost: 20, percentOfWeapon: 50},
}

// holyStrikeBonusCoefficient is every rank's sp_coefficient (constant
// across all eight ranks in the client data).
const holyStrikeBonusCoefficient = 0.429

// holyStrikeSacredArbiterDamageMultiplier: source 1.60.1.70009 client
// talent data (talents/paladin.json, Retribution tree,
// "Sacred Arbiter", 1 rank, spell id 1311087): "Increases the damage of
// your Holy Strike ability by 20% and causes it to refresh all Judgement
// effects on the target."
//
// "Improved Holy Strike" does not exist anywhere in this client's talent
// trees, and the regenerated proto no longer carries a field for it.
const holyStrikeSacredArbiterDamageMultiplier = 1.2

func (paladin *Paladin) registerHolyStrike() {
	cd := core.Cooldown{
		Timer:    paladin.NewTimer(),
		Duration: time.Second * 10,
	}

	hasSacredArbiter := paladin.Talents.SacredArbiter

	for i, rank := range holyStrikeRanks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		flatDamage := HolyStrikeFlatDamage[i+1]
		casterLevel := int(paladin.Level)

		damageMultiplier := rank.percentOfWeapon / 100
		if hasSacredArbiter {
			damageMultiplier *= holyStrikeSacredArbiterDamageMultiplier
		}

		paladin.RegisterSpell(core.SpellConfig{
			SpellCode:      SpellCode_PaladinHolyStrike,
			ClassSpellMask: PaladinSpellMaskHolyStrike,
			ActionID:       core.ActionID{SpellID: rank.spellID},
			SpellSchool:    core.SpellSchoolHoly,
			DefenseType:    core.DefenseTypeMelee,
			ProcMask:       core.ProcMaskMeleeMHSpecial,
			Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

			RequiredLevel: int(rank.level),
			Rank:          i + 1,

			ManaCost: core.ManaCostOptions{
				FlatCost: rank.manaCost,
			},
			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD: core.GCDDefault,
				},
				CD: cd,
			},

			DamageMultiplier: damageMultiplier,
			ThreatMultiplier: 1,
			BonusCoefficient: holyStrikeBonusCoefficient,
			ClientBaseDamage: flatDamage.Range(casterLevel),

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				baseDamage := flatDamage.Roll(sim, casterLevel) + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
				result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

				if hasSacredArbiter && result.Landed() {
					for _, judgementAura := range target.GetAurasWithTag(core.JudgementAuraTag) {
						if judgementAura.IsActive() {
							judgementAura.Refresh(sim)
						}
					}
				}
			},
		})
	}
}
