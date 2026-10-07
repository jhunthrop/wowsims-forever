package druid

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const MoonfireRanks = 10

var MoonfireSpellId = [MoonfireRanks + 1]int32{0, 8921, 8924, 8925, 8926, 8927, 8928, 8929, 9833, 9834, 9835}

// The direct hit's damage and coefficient and the DoT's per-tick damage and
// coefficient are spellconst/druid.json's own (rank 10: 135 at 0.15 up
// front, then 60 at 0.13 on each of four 3 s ticks, 240 in all, against the
// Classic 195-228 and 384 these replaced).
var MoonfireSpellCoeff = [MoonfireRanks + 1]float64{0, .15, .15, .15, .15, .15, .15, .15, .15, .15, .15}
var MoonfireDotSpellCoeff = [MoonfireRanks + 1]float64{0, .13, .13, .13, .13, .13, .13, .13, .13, .13, .13}
var MoonfireBaseDamage = [MoonfireRanks + 1]float64{0, 8, 13, 22, 32, 47, 59, 73, 91, 111, 135}
var MoonfireTickDamage = [MoonfireRanks + 1]float64{0, 4, 6, 9, 13, 19, 23, 31, 39, 49, 60}
var MoonfireDotTicks = [MoonfireRanks + 1]int32{0, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4}
var MoonfireManaCost = [MoonfireRanks + 1]float64{0, 25, 50, 75, 105, 150, 190, 235, 280, 325, 375}
var MoonfireLevel = [MoonfireRanks + 1]int{0, 4, 10, 16, 22, 28, 34, 40, 46, 52, 58}

func (druid *Druid) registerMoonfireSpell() {
	druid.Moonfire = make([]*DruidSpell, 0)

	for rank := 1; rank <= MoonfireRanks; rank++ {
		config := druid.getMoonfireBaseConfig(rank)

		if config.RequiredLevel <= int(druid.Level) {
			druid.Moonfire = append(druid.Moonfire, druid.RegisterSpell(Humanoid|Moonkin, config))
		}
	}
}

func (druid *Druid) getMoonfireBaseConfig(rank int) core.SpellConfig {
	ticks := MoonfireDotTicks[rank]
	tickLength := time.Second * 3

	spellId := MoonfireSpellId[rank]
	spellCoeff := MoonfireSpellCoeff[rank]
	spellDotCoeff := MoonfireDotSpellCoeff[rank]
	baseDamage := MoonfireBaseDamage[rank]
	baseDotDamage := MoonfireTickDamage[rank]
	manaCost := MoonfireManaCost[rank]
	level := MoonfireLevel[rank]

	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: spellId},
		SpellCode:      SpellCode_DruidMoonfire,
		ClassSpellMask: DruidSpellMaskMoonfire,
		SpellSchool:    core.SpellSchoolArcane,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          SpellFlagOmen | core.SpellFlagAPL | core.SpellFlagResetAttackSwing,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: 0,
			},
		},
		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:    fmt.Sprintf("Moonfire (Rank %d)", rank),
				ActionID: core.ActionID{SpellID: spellId},
			},
			NumberOfTicks:    ticks,
			TickLength:       tickLength,
			BonusCoefficient: spellDotCoeff,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDotDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		BonusCoefficient: spellCoeff,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			if result.Landed() {
				dot := spell.Dot(target)
				dot.Apply(sim)
			}
		},
	}
}
