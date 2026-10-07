package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

type RipRankInfo struct {
	id              int32
	level           int32
	dmgTickBase     float64
	dmgTickPerCombo float64
}

var ripRanks = []RipRankInfo{
	{
		id:              1079,
		level:           20,
		dmgTickBase:     3.0,
		dmgTickPerCombo: 4.0,
	},
	{
		id:              9492,
		level:           28,
		dmgTickBase:     4.0,
		dmgTickPerCombo: 7.0,
	},
	{
		id:              9493,
		level:           36,
		dmgTickBase:     6.0,
		dmgTickPerCombo: 9.0,
	},
	{
		id:              9752,
		level:           44,
		dmgTickBase:     9.0,
		dmgTickPerCombo: 14.0,
	},
	{
		id:              9894,
		level:           52,
		dmgTickBase:     12.0,
		dmgTickPerCombo: 20.0,
	},
	{
		id:              9896,
		level:           60,
		dmgTickBase:     17.0,
		dmgTickPerCombo: 28.0,
	},
}

// RipBaseTicks/RipTicks/RipDuration: Classic's well-documented Rip
// duration scales with combo points the same way Rogue's Rupture does
// (sim/rogue/rupture.go's RuptureTicks): 3 ticks @ 2s (6s) base, +1 tick
// (2s) per combo point, giving 8/10/12/14/16 sec for 1-5 combo points.
// The client's duration_ms (12000ms, identical on every one of Rip's six
// ranks in spellconst) carries no combo-point-scaling effect field --
// unlike an ability whose client data ties a term to a misc_value, this
// number is a server-side-script placeholder the client alone can't
// source, so per this lane's rule for that case Classic's own published
// formula is kept instead, and the fixed six ticks (12s regardless of
// combo points) the engine had is the bug this closes: it made
// Ferocious Bite (which reads whether Rip is about to expire)
// unreachable, since Rip never got close to expiring at low combo point
// counts the way it should.
const RipBaseTicks int32 = 3

func (druid *Druid) RipTicks(comboPoints int32) int32 {
	return RipBaseTicks + comboPoints
}

func (druid *Druid) RipDuration(comboPoints int32) time.Duration {
	return time.Duration(druid.RipTicks(comboPoints)) * time.Second * 2
}

func (druid *Druid) registerRipSpell() {
	// Add highest available Rip rank for level.
	for rank := len(ripRanks) - 1; rank >= 0; rank-- {
		if druid.Level >= ripRanks[rank].level {
			config := druid.newRipSpellConfig(ripRanks[rank])
			druid.Rip = druid.RegisterSpell(Cat, config)
			return
		}
	}
}

func (druid *Druid) newRipSpellConfig(ripRank RipRankInfo) core.SpellConfig {
	energyCost := 30.0

	return core.SpellConfig{
		SpellCode:      SpellCode_DruidRip,
		ClassSpellMask: DruidSpellMaskRip,
		ActionID:       core.ActionID{SpellID: ripRank.id},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          SpellFlagOmen | core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagPureDot,

		RequiredLevel: int(ripRank.level),

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

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rip",
			},
			NumberOfTicks: 0, // Set dynamically, by combo points, in ApplyEffects.
			TickLength:    time.Second * 2,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				cp := float64(druid.ComboPoints())
				cpScaling := core.TernaryFloat64(cp == 5, 4, cp)
				baseDamage := ripRank.dmgTickBase + ripRank.dmgTickPerCombo*cp
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
				dot.NumberOfTicks = druid.RipTicks(druid.ComboPoints())
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
