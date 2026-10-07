package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Lacerate (spells 24118, 24119, 24120 and 1299332, learned at 30, 40, 50
// and 60): "Wounds the target causing them to bleed for N damage over 21
// sec." Requires a melee weapon. The client's effect is a periodic damage
// aura (effect 6, aura 3) of a per-tick amount every 3 seconds for 21
// seconds, with no attack power coefficient. This is the trainable
// ability, not the Lacerating Strikes talent (lacerating_strikes.go),
// which is a bleed on Mongoose Bite.
const (
	lacerateTicks      = 7
	lacerateTickLength = 3 * time.Second
)

// lacerateLearnLevels are the four ranks' learn levels, read from the
// generated LacerateLevel column.
var lacerateLearnLevels = LacerateLevel[1:]

func (hunter *Hunter) getLacerateConfig(rank int) core.SpellConfig {
	damage := LacerateDamage[rank]
	casterLevel := int(hunter.Level)

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterLacerate,
		ActionID:      core.ActionID{SpellID: LacerateSpellId[rank]},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagNoOnCastComplete,
		Rank:          rank,
		RequiredLevel: LacerateLevel[rank],

		ManaCost: core.ManaCostOptions{
			FlatCost: LacerateManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: damage.Range(casterLevel),

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Lacerate",
				Tag:   "Lacerate",
			},
			NumberOfTicks: lacerateTicks,
			TickLength:    lacerateTickLength,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, damage.Roll(sim, casterLevel), isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
			if result.Landed() {
				spell.Dot(target).Apply(sim)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},
	}
}

func (hunter *Hunter) registerLacerateSpell() {
	rank := core.HighestRankAtLevel(lacerateLearnLevels, hunter.Level)
	if rank == 0 {
		return
	}
	hunter.Lacerate = hunter.GetOrRegisterSpell(hunter.getLacerateConfig(rank))
}
