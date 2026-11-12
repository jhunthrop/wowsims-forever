package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// RipTickDamage is constants_auto_gen.go's row as an Effect per rank.
var RipTickDamage = clientdamage.FromTable(RipBaseDamage[:], RipPointsPerLevel[:], RipLevel[:], RipMaxLevel[:])

// The Rip ladder is constants_auto_gen.go's: RipBaseDamage is the client's
// per-tick base (rank 6: 19 after the 70755 push 113403, 15 before it).
// ripTickPerComboPoint is the client's EffectPointsPerResource for effect 0
// of each rank (ids 1079-9896; 1.60.1.70755 server push 113403), which the
// vendored spellconst does not carry: the tooltip reads "${6*($m1+N*$b1)}
// damage over $d" for N combo points, so each combo point adds its step to
// every one of the six ticks.
var ripTickPerComboPoint = [RipRanks + 1]float64{0, 5.5, 9, 10.5, 15.8, 22.5, 31.9}

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
				baseDamage := tickDamage.Center(casterLevel) + tickPerComboPoint*cp
				tickDamage := baseDamage + ripTickAttackPowerShare(cp)*dot.Spell.MeleeAttackPower(target)
				// Rend and Tear on the bleed when the target already bled as it
				// was applied (bleedRendAndTear, feral_talents.go).
				dot.Snapshot(target, tickDamage*druid.bleedRendAndTear, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.BreakProwl(sim)

			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
			if result.Landed() {
				druid.bleedRendAndTear = druid.RendAndTearMultiplier(target)
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

// ripAttackPowerPerPointTick is Rip's share of attack power a combo point on
// each tick. Era's was 1% (6% a point over the six ticks, four points at
// most); Blizzard's Forever notes of 3 November 2026 raise Rip's "base points
// and attack power scaling ... by about 25%", and the base half matches the
// client's push (15 to 19 a tick at rank 6), so the share is 1.25%.
const ripAttackPowerPerPointTick = 0.0125

// ripTickAttackPowerShare is the attack power share of one Rip tick at the
// given combo points: the fifth point adds no attack power.
func ripTickAttackPowerShare(comboPoints float64) float64 {
	return ripAttackPowerPerPointTick * min(comboPoints, 4)
}
