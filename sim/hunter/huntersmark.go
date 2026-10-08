package hunter

import (
	"github.com/wowsims/classic/sim/core"
)

// huntersMarkRangedAttackPower is each rank's ranged attack power, indexed
// like the generated HunterSMark tables: the client's effect 1 (aura 127,
// ranged attack power against the target) base points of spells 1130,
// 14323, 14324 and 1213268 (1.60.1.70009 SpellEffect.csv). The generated
// BaseDamage row of this spell is a dummy, so the amounts live here, pinned
// by TestHuntersMarkRanksMatchClient.
var huntersMarkRangedAttackPower = [HunterSMarkRanks + 1]float64{0, 26, 59, 98, 110}

func (hunter *Hunter) registerHuntersMarkSpell() {
	rank := core.HighestRankAtLevel(HunterSMarkLevel[1:], hunter.Level)
	if rank == 0 {
		return
	}

	spellID := HunterSMarkSpellId[rank]
	bonus := huntersMarkRangedAttackPower[rank]
	auras := hunter.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.HuntersMarkAuraOfRank(target, spellID, bonus, 0)
	})

	hunter.HuntersMark = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_HunterHuntersMark,
		ActionID:      core.ActionID{SpellID: spellID},
		SpellSchool:   core.SpellSchoolArcane,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
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
