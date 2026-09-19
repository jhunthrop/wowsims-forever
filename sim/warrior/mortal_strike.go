package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Mortal Strike's generated rows are two of the three the data lane owes
// a fix for, so its numbers are the only ones in this file still typed:
//
//   - MortalStrikeBaseDamage is {-50,-50} at every rank. Spell 27580's
//     first effect is the -50% healing-taken aura (effect 6, aura 118)
//     and the generator emits a spell's school-damage effect or, failing
//     that, its first; Mortal Strike's damage is effect 121 ("weapon
//     damage plus 160" at rank 4), which the generated arrays do not
//     carry at all.
//   - MortalStrikeManaCost is 0 at rank 4. Two rank-4 rows share
//     spell_level 60 - 21553 at cost 300 and 27580 at cost 0 - and the
//     dedup broke the tie on the higher id, so it kept the free 27580.
//
// Both figures below are therefore read by hand from
// data/builds/1.60.1.69893/spellconst/warrior.json: 160 is the effect
// 121 amount both rank-4 rows carry, and 30 rage is the 300-tenths cost
// of 21553, the row the engine keeps. The cooldown is read from the
// generated array, which is sound.
const (
	mortalStrikeBonusDamage = 160.0
	mortalStrikeRageCost    = 30.0
)

func (warrior *Warrior) registerMortalStrikeSpell(cdTimer *core.Timer) {
	if !warrior.Talents.MortalStrike {
		return
	}

	rank := rankAtLevel(MortalStrikeLevel[:], warrior.Level)
	// The engine keeps spell 21553 rather than the generated
	// MortalStrikeSpellId[4] of 27580: the two are the same rank-4
	// Mortal Strike, 21553 is the one that carries the client's 300
	// cost, and it is the id the UI and the preset rotations name.
	// Swapping the ids is the data lane's call, not this file's.
	spellID := int32(21553)

	warrior.MortalStrike = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorMortalStrike,
		ClassSpellMask: WarriorSpellMaskMortalStrike,
		ActionID:       core.ActionID{SpellID: spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: MortalStrikeLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   mortalStrikeRageCost,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: time.Duration(MortalStrikeCooldownMS[rank]) * time.Millisecond,
			},
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := mortalStrikeBonusDamage + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
