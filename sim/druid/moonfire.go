package druid

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
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
var MoonfireDamage = [MoonfireRanks + 1]clientdamage.Effect{
	{},
	{Amount: 8, Variance: 0.25, PerLevel: 0.5, SpellLevel: 4, MaxLevel: 9},
	{Amount: 13, Variance: 0.266667, PerLevel: 0.8, SpellLevel: 10, MaxLevel: 15},
	{Amount: 22, Variance: 0.214286, PerLevel: 1, SpellLevel: 16, MaxLevel: 21},
	{Amount: 32, Variance: 0.181818, PerLevel: 1.5, SpellLevel: 22, MaxLevel: 27},
	{Amount: 47, Variance: 0.179104, PerLevel: 1.5, SpellLevel: 28, MaxLevel: 33},
	{Amount: 59, Variance: 0.179775, PerLevel: 1.6, SpellLevel: 34, MaxLevel: 39},
	{Amount: 73, Variance: 0.173913, PerLevel: 1.7, SpellLevel: 40, MaxLevel: 45},
	{Amount: 91, Variance: 0.169014, PerLevel: 2.7, SpellLevel: 46, MaxLevel: 51},
	{Amount: 111, Variance: 0.163743, PerLevel: 2.1, SpellLevel: 52, MaxLevel: 57},
	{Amount: 135, Variance: 0.156098, PerLevel: 2.3, SpellLevel: 58, MaxLevel: 63},
}
var MoonfireTickDamage = [MoonfireRanks + 1]clientdamage.Effect{
	{},
	{Amount: 4, SpellLevel: 4, MaxLevel: 9},
	{Amount: 6, SpellLevel: 10, MaxLevel: 15},
	{Amount: 9, SpellLevel: 16, MaxLevel: 21},
	{Amount: 13, SpellLevel: 22, MaxLevel: 27},
	{Amount: 19, SpellLevel: 28, MaxLevel: 33},
	{Amount: 23, SpellLevel: 34, MaxLevel: 39},
	{Amount: 31, SpellLevel: 40, MaxLevel: 45},
	{Amount: 39, SpellLevel: 46, MaxLevel: 51},
	{Amount: 49, SpellLevel: 52, MaxLevel: 57},
	{Amount: 60, SpellLevel: 58, MaxLevel: 63},
}
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
	damage := MoonfireDamage[rank]
	tickDamage := MoonfireTickDamage[rank]
	casterLevel := int(druid.Level)
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
				dot.Snapshot(target, tickDamage.Center(casterLevel), isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		BonusCoefficient: spellCoeff,
		ClientBaseDamage: damage.Range(casterLevel),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)

			if result.Landed() {
				dot := spell.Dot(target)
				dot.Apply(sim)
			}
		},
	}
}
