package hunter

import (
	"github.com/wowsims/classic/sim/core"
)

// huntersMarkRangedAttackPower is each rank's ranged attack power, indexed
// like the generated HunterSMark tables: the client's effect 1 (aura 127,
// ranged attack power against the target) base points of spells 1130,
// 14323, 14324 and 14325 (1.60.1.70707 SpellEffect.csv). The generated
// BaseDamage row of this spell is a dummy, so the amounts live here, pinned
// by TestHuntersMarkRanksMatchClient.
var huntersMarkRangedAttackPower = [HunterSMarkRanks + 1]float64{0, 26, 59, 98, core.HuntersMarkTopRankRangedAttackPower}

// huntersMarkSpellID reads a rank's id, correcting rank 4: the generator
// kept 1213268, a Season of Discovery copy at 110 that no trainer teaches,
// where SkillLineAbility (row 7913) teaches 14325, 143 since the 23 October
// hotfix and in the 1.60.1.70707 tables.
func huntersMarkSpellID(rank int) int32 {
	if rank == HunterSMarkRanks {
		return core.HuntersMarkTopRankSpellID
	}
	return HunterSMarkSpellId[rank]
}

func (hunter *Hunter) registerHuntersMarkSpell() {
	rank := core.HighestRankAtLevel(HunterSMarkLevel[1:], hunter.Level)
	if rank == 0 {
		return
	}

	spellID := huntersMarkSpellID(rank)
	bonus := huntersMarkRangedAttackPower[rank]
	auras := hunter.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.HuntersMarkAuraOfRank(target, spellID, bonus, 0)
	})

	hunter.HuntersMark = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_HunterHuntersMark,
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolArcane,
		DefenseType: core.DefenseTypeMagic,
		// The mark deals no damage. Without a damage mask the pre-pull
		// landing hold (core.prepullLandingHold) leaves it where the
		// rotation puts it, before the pull, instead of holding it to 0 and
		// spending the opening global (sim-accuracy round 21).
		ProcMask:      core.ProcMaskEmpty,
		Flags:         core.SpellFlagAPL,
		Rank:          rank,
		RequiredLevel: HunterSMarkLevel[rank],

		ManaCost: core.ManaCostOptions{
			FlatCost: HunterSMarkManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true, // Hunter GCD is locked at 1.5s
		},

		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				auras.Get(target).Activate(sim)
			}
		},

		RelatedAuras: []core.AuraArray{auras},
	})
}
