package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// rendTicks is how many three-second ticks each rank's bleed runs for.
// The generated arrays carry no duration column, so it is read by hand
// from the client's own duration_ms / period_ms
// (data/builds/1.60.1.69893/spellconst/warrior.json): 9/12/15/18/21/21/21
// seconds over a 3 s period. Listed for the data lane as the one column
// this ability still cannot read.
var rendTicks = [RendRanks + 1]int32{0, 3, 4, 5, 6, 7, 7, 7}

func (warrior *Warrior) registerRendSpell() {
	rank := rankAtLevel(RendLevel[:], warrior.Level)
	baseDamage := RendBaseDamage[rank][0]

	// Improved Rend: "Increases the Bleed damage done by your Rend
	// ability by 12%" at rank 1, "by 23%" at rank 2 and "by 35%" at
	// rank 3 (Arms node 105956). Vanilla's 15/25/35 is what stood here;
	// only the top rank happens to agree.
	damageMultiplier := []float64{1, 1.12, 1.23, 1.35}[warrior.Talents.ImprovedRend]

	warrior.Rend = warrior.RegisterSpell(BattleStance|DefensiveStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorRend,
		ClassSpellMask: WarriorSpellMaskRend,
		ActionID:       core.ActionID{SpellID: RendSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagAPL | core.SpellFlagNoOnCastComplete | SpellFlagOffensive,

		RequiredLevel: RendLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   rageCost(RendManaCost[rank]),
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		DamageMultiplier: damageMultiplier,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rend",
				Tag:   "Rend",
			},
			NumberOfTicks: rendTicks[rank],
			TickLength:    time.Second * 3,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
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
	})

}
