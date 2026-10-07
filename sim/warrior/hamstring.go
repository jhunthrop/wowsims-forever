package warrior

import (
	"github.com/wowsims/classic/sim/core"
)

// hamstringRankSpellID reads a Hamstring rank's id, correcting the one
// rank where the generator's dedup kept the wrong duplicate: rank 3's
// real, player-cast row (7373, spell_level 54, cost 100) and a free
// reissue (27584, same spell_level, cost 0) tie on spell_level, and the
// generator's tie-break kept 27584 in HamstringSpellId[3]. 7373 is the
// id the preset rotations name, and the one this file hardcoded for
// EVERY rank until now - which mislabeled ranks 1-2 and compared them
// against rank 3's spell_level (54) instead of their own (8, 32).
func hamstringRankSpellID(rank int) int32 {
	if rank == 3 {
		return 7373
	}
	return HamstringSpellId[rank]
}

func (warrior *Warrior) registerHamstringSpell() {
	rank := rankAtLevel(HamstringLevel[:], warrior.Level)
	damage := HamstringBaseDamage[rank][0]
	spellID := hamstringRankSpellID(rank)
	spell_level := float64(HamstringLevel[rank])

	// HamstringManaCost[3] is 0 and is NOT read here: two rank-3 rows
	// share spell_level 54 - 7373 at cost 100 and 27584 at cost 0 - and
	// the dedup kept the free 27584, so the cost below is read by hand
	// from 7373's own 100 tenths. Listed for the data lane.
	const hamstringRageCost = 10.0

	warrior.Hamstring = warrior.RegisterSpell(BattleStance|BerserkerStance, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: spellID},
		ClassSpellMask: WarriorSpellMaskHamstring,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagBinary | SpellFlagOffensive,

		RequiredLevel: HamstringLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   hamstringRageCost,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1.25,
		FlatThreatBonus:  1.25 * 2 * float64(spell_level),
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
