package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// laceratingStrikesTicks and laceratingStrikesTickLength are Lacerating
// Strikes' bleed shape; source: 1.60.1.70009 client spell data (spell
// 1310536: duration_ms 21000, period_ms 3000 -- 21s / 3s = 7 ticks). The
// talent text (spell 1310533) gives the total: "causes the target to
// Bleed for damage over 21 sec equal to 40% of the damage done by
// Mongoose Bite" -- so each tick is that 40% split evenly across the 7
// ticks, the same shape sim/warrior/deep_wounds.go uses for its own
// percent-of-a-hit bleed.
const laceratingStrikesTicks = 7
const laceratingStrikesTickLength = time.Second * 3
const laceratingStrikesPercentOfHit = 0.4

// registerLaceratingStrikesDot registers the Lacerating Strikes bleed as
// its own spell/dot, mirroring sim/warrior/deep_wounds.go's pattern for
// a percent-of-a-triggering-hit bleed: no ActionID damage of its own,
// just a periodic tick, triggered from Mongoose Bite's ApplyEffects
// rather than from a cast the player chooses.
func (hunter *Hunter) registerLaceratingStrikesDot() {
	if !hunter.Talents.LaceratingStrikes {
		return
	}

	hunter.LaceratingStrikes = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_HunterLaceratingStrikes,
		ActionID:    core.ActionID{SpellID: 1310533},
		SpellSchool: core.SpellSchoolPhysical,
		// A critting tick asks the spell for its crit multiplier, which
		// needs a DefenseType to pick the melee crit bonus.
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Lacerating Strikes",
			},
			NumberOfTicks: laceratingStrikesTicks,
			TickLength:    laceratingStrikesTickLength,
			// Forever's 24 September 2026 beta notes: "Lacerating
			// Strikes: can now land critical hits." Each tick rolls the
			// hunter's melee crit, the same per-tick physical roll the
			// engine's other critting bleeds use.
			CanCrit: true,

			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTickPhysicalCrit)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Dot(target).Apply(sim)
		},
	})
}

// tryProcLaceratingStrikes applies (or refreshes) the Lacerating Strikes
// bleed off a landed Mongoose Bite hit, snapshotting 40% of that hit's
// damage split evenly across the bleed's 7 ticks.
func (hunter *Hunter) tryProcLaceratingStrikes(sim *core.Simulation, target *core.Unit, result *core.SpellResult) {
	if hunter.LaceratingStrikes == nil || !result.Landed() {
		return
	}

	dot := hunter.LaceratingStrikes.Dot(target)
	dot.SnapshotBaseDamage = result.Damage * laceratingStrikesPercentOfHit / laceratingStrikesTicks
	dot.SnapshotAttackerMultiplier = 1

	hunter.LaceratingStrikes.Cast(sim, target)
}
