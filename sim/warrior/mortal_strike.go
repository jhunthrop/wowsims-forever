package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Mortal Strike's generated rows are two of the three the data lane owes
// a fix for, so two things stay typed here:
//
//   - MortalStrikeBaseDamage is {-50,-50} at every rank: spell 27580's
//     first effect is the -50% healing-taken aura (effect 6, aura 118)
//     and the generator emits a spell's school-damage effect or, failing
//     that, its first. Mortal Strike's damage is effect 121 ("weapon
//     damage plus <N>"), which MortalStrikeDamage (client_damage.go)
//     states from the client file and spellconst_damage_test.go checks.
//   - MortalStrikeManaCost is 0 at rank 4. Two rank-4 rows share
//     spell_level 60 - 21553 at cost 300 and 27580 at cost 0 - and the
//     dedup broke the tie on the higher id, so it kept the free 27580.
//     The 300-tenths cost (30 rage at every rank) is read by hand.
const mortalStrikeRageCost = 30.0

func (warrior *Warrior) registerMortalStrikeSpell(cdTimer *core.Timer) {
	if !warrior.Talents.MortalStrike {
		return
	}

	rank := rankAtLevel(MortalStrikeLevel[:], warrior.Level)
	bonusDamage := MortalStrikeDamage[rank]
	casterLevel := int(warrior.Level)
	// The engine keeps spell 21553 rather than the generated
	// MortalStrikeSpellId[4] of 27580 for rank 4 only: the two are the
	// same rank-4 Mortal Strike, 21553 is the one that carries the
	// client's 300 cost, and it is the id the UI and the preset
	// rotations name. Swapping THAT id is the data lane's call, not
	// this file's. Ranks 1-3 (12294/21551/21552) have no such
	// duplicate and are registered under the generated id, so a
	// levelling warrior's Mortal Strike resolves at every rank rather
	// than only at 60.
	spellID := MortalStrikeSpellId[rank]
	if rank == MortalStrikeRanks {
		spellID = 21553
	}

	castConfig := core.CastConfig{
		DefaultCast: core.Cast{
			GCD: core.GCDDefault,
		},
		IgnoreHaste: true,
	}
	// Rank 0 - a warrior below MortalStrikeLevel[1]=40 with the talent
	// already spent - carries a zero cooldown; guard as slam.go does.
	if cooldownMS := MortalStrikeCooldownMS[rank]; cooldownMS > 0 {
		castConfig.CD = core.Cooldown{
			Timer:    cdTimer,
			Duration: time.Duration(cooldownMS) * time.Millisecond,
		}
	}

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
		Cast: castConfig,

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,
		ClientBaseDamage: bonusDamage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := bonusDamage.Roll(sim, casterLevel) + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
