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
// damage) gives flatBonus, a flat Holy bonus beside the normalized main
// hand weapon damage, plus spell damage scaling via sp_coefficient (a
// constant 0.429 across all eight ranks, see holyStrikeBonusCoefficient,
// the same coefficient Exorcism uses); index 1 (effect 31, weapon percent
// damage) gives percentOfWeapon, the share of the normalized weapon damage
// the strike deals; index 2 (effect 77, script effect) has no base_points
// at any rank (always 0) and is the server-side hook Sacred Arbiter's
// refresh-Judgement behaviour hangs on, not a value this engine reads.
//
// The percent scales the weapon damage only. The flat bonus and the
// spell damage term are added whole, as the client's text says ("causes
// 50% weapon damage plus an additional 81 to 105 as Holy damage"), and as
// the live logs show (sim-accuracy lane sim-paladins, 11 Nov 2026): a
// level 30 to 35 Retribution paladin's 78 rank 4 hits averaged 106 against
// his own pre-armor white swings of 198 and Seal of Command procs of 145,
// which is 0.36 x 202 + 25 (98, 107 with his Judgement of the Crusader on
// a quarter of them) and not (202 + 25) x 0.36 (82); two paladins on
// foreverlogs.gg landed Holy Strike at 0.73 to 0.75 of a Seal of Command
// proc, against 0.60 for the old reading. Until then this file applied
// the percent as the spell's DamageMultiplier over flat and weapon alike,
// the way Backstab's 150% multiplies its own flat, which halved the flat
// and the spell damage at 60 and quartered them at rank 1.
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

// Shieldsworn (node 110875, Protection tier 6 col 1, spell 1328544, client
// 1.60.1.70707): "Reduces the cooldown of your Holy Strike ability by 4
// sec. Additionally, Holy Strike now strikes with your shield, triggering
// your active Holy Shield and Shield Specialization talents without
// consuming their charges or triggering their cooldowns, and allowing you
// to Block spells for 6 sec." The strike fires Holy Shield's block damage
// while Holy Shield is up and rolls Shield Specialization's mana without
// its 3 second internal cooldown, both only with a shield on (it strikes
// with the shield). The spell block is not modelled: the tank boss's
// damage is melee.
const (
	holyStrikeCooldown               = 10 * time.Second
	shieldswornHolyStrikeCooldownCut = 4 * time.Second
)

func (paladin *Paladin) holyStrikeCooldown() time.Duration {
	if paladin.Talents.Shieldsworn {
		return holyStrikeCooldown - shieldswornHolyStrikeCooldownCut
	}
	return holyStrikeCooldown
}

// strikeWithShield is Shieldsworn's shield half of a landed Holy Strike.
func (paladin *Paladin) strikeWithShield(sim *core.Simulation, target *core.Unit) {
	if !paladin.Talents.Shieldsworn || !paladin.PseudoStats.CanBlock {
		return
	}
	if i := paladin.activeHolyShield(); i >= 0 {
		paladin.holyShieldProc[i].Cast(sim, target)
	}
	if paladin.shieldSpecializationOnStrike != nil {
		paladin.shieldSpecializationOnStrike(sim)
	}
}

func (paladin *Paladin) registerHolyStrike() {
	cd := core.Cooldown{
		Timer:    paladin.NewTimer(),
		Duration: paladin.holyStrikeCooldown(),
	}

	hasSacredArbiter := paladin.Talents.SacredArbiter

	for i, rank := range holyStrikeRanks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		flatDamage := HolyStrikeFlatDamage[i+1]
		casterLevel := int(paladin.Level)

		weaponShare := rank.percentOfWeapon / 100
		damageMultiplier := 1.0
		if hasSacredArbiter {
			damageMultiplier = holyStrikeSacredArbiterDamageMultiplier
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
				FlatCost:   rank.manaCost,
				Multiplier: paladin.benediction(),
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
				baseDamage := flatDamage.Roll(sim, casterLevel) + weaponShare*spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
				result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
				// Iron Creed's proc (1311034, aura 231 on a done melee-class
				// spell, no hit mask) needs a landed hit or crit.
				if result.Landed() {
					paladin.activateIronCreed(sim)
					paladin.strikeWithShield(sim, target)
				}

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
