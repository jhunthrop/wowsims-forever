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
//     damage plus <N>" per rank), which the generated arrays do not
//     carry at all.
//   - MortalStrikeManaCost is 0 at rank 4. Two rank-4 rows share
//     spell_level 60 - 21553 at cost 300 and 27580 at cost 0 - and the
//     dedup broke the tie on the higher id, so it kept the free 27580.
//
// Both figures below are therefore read by hand, per rank, from
// data/builds/1.60.1.70009/spellconst/warrior.json's effect-121 amount
// (85/110/135/160 at ranks 1-4, ids 12294/21551/21552/21553) and its
// 300-tenths cost (30 rage at every rank). The cooldown is read from
// the generated array, which is sound. Index 0 is the unused phantom
// rank rankAtLevel never returns once the MortalStrike talent's own
// level-40 floor is respected.
var mortalStrikeBonusDamageByRank = [MortalStrikeRanks + 1]float64{0, 85, 110, 135, 160}

const mortalStrikeRageCost = 30.0

func (warrior *Warrior) registerMortalStrikeSpell(cdTimer *core.Timer) {
	if !warrior.Talents.MortalStrike {
		return
	}

	rank := rankAtLevel(MortalStrikeLevel[:], warrior.Level)
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

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := mortalStrikeBonusDamageByRank[rank] + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
