package warrior

import (
	"github.com/wowsims/classic/sim/core"
)

func (warrior *Warrior) registerHamstringSpell() {
	rank := rankAtLevel(HamstringLevel[:], warrior.Level)
	damage := HamstringBaseDamage[rank][0]
	// The engine keeps spell 7373, the id the preset rotations name; the
	// generated HamstringSpellId[3] is 27584, the reissue the dedup kept
	// over it. Same rank, same 45 damage.
	spellID := int32(7373)
	spell_level := float64(HamstringLevel[rank])

	// HamstringManaCost[3] is 0 and is NOT read here: two rank-3 rows
	// share spell_level 54 - the 100-tenths 7373/27584 and a free
	// reissue - and the dedup kept the free one, so the cost below is
	// read by hand from the client's own 100. Listed for the data lane.
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
