package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// RipTickDamage is constants_auto_gen.go's row as an Effect per rank.
var RipTickDamage = clientdamage.FromTable(RipBaseDamage[:], RipPointsPerLevel[:], RipLevel[:], RipMaxLevel[:])

// The Rip ladder is constants_auto_gen.go's: RipBaseDamage is the client's
// per-tick base (rank 6: 15, where the Era ladder this replaced had 17).
// ripTickPerComboPoint is the client's EffectPointsPerResource for effect 0
// of each rank (1.60.1.70009 SpellEffect.csv, ids 1079-9896), which the
// vendored spellconst does not carry: the tooltip reads "${6*($m1+N*$b1)}
// damage over $d" for N combo points, so each combo point adds its step to
// every one of the six ticks.
var ripTickPerComboPoint = [RipRanks + 1]float64{0, 4.4, 7.2, 8.5, 12.7, 18.2, 25.5}

// RipNumberOfTicks is Rip's tick count at every combo point count: the
// client's duration_ms is 12000 on all six ranks with 2 s periods, and its
// tooltip states "damage over $d" for 1 through 5 points alike (unlike
// Rupture, whose tooltip lists 8/10/12/14/16 secs). Combo points scale the
// damage per tick, never the duration.
const RipNumberOfTicks int32 = 6

func (druid *Druid) registerRipSpell() {
	// Add highest available Rip rank for level.
	for rank := RipRanks; rank >= 1; rank-- {
		if druid.Level >= int32(RipLevel[rank]) {
			config := druid.newRipSpellConfig(rank)
			druid.Rip = druid.RegisterSpell(Cat, config)
			return
		}
	}
}

func (druid *Druid) newRipSpellConfig(rank int) core.SpellConfig {
	energyCost := 30.0
	tickDamage := RipTickDamage[rank]
	casterLevel := int(druid.Level)
	tickPerComboPoint := ripTickPerComboPoint[rank]

	return core.SpellConfig{
		SpellCode:      SpellCode_DruidRip,
		ClassSpellMask: DruidSpellMaskRip,
		ActionID:       core.ActionID{SpellID: RipSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          SpellFlagOmen | core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagPureDot,

		RequiredLevel: RipLevel[rank],
		Rank:          rank,

		EnergyCost: core.EnergyCostOptions{
			Cost:   energyCost,
			Refund: 0,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return druid.ComboPoints() > 0
		},

		ClientBaseDamage: tickDamage.Range(casterLevel),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rip",
			},
			NumberOfTicks: RipNumberOfTicks,
			TickLength:    time.Second * 2,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				cp := float64(druid.ComboPoints())
				cpScaling := core.TernaryFloat64(cp == 5, 4, cp)
				baseDamage := tickDamage.Center(casterLevel) + tickPerComboPoint*cp
				// AP scaling is 6% per combo point from 1 to 4, and 24% again for 5
				tickDamage := baseDamage + 0.01*cpScaling*dot.Spell.MeleeAttackPower(target)
				dot.Snapshot(target, tickDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.BreakProwl(sim)

			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
			if result.Landed() {
				dot := spell.Dot(target)
				dot.NumberOfTicks = RipNumberOfTicks + dot.ModNumberOfTicks
				dot.Apply(sim)
				druid.SpendComboPoints(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},
	}
}

func (druid *Druid) CurrentRipCost() float64 {
	return druid.Rip.Cost.GetCurrentCost()
}
