package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// ruptureLearnLevels are Rupture's six rank learn levels; source:
// 1.60.1.70009 client spell data ("Rupture", ranks 1-6; the level-20
// rank-0 ids are internal copies, not player ranks).
var ruptureLearnLevels = []int{20, 28, 36, 44, 52, 60}

// ruptureSpellID is Rupture's rank -> spell id, index 0 unused.
var ruptureSpellID = [7]int32{0, 1943, 8639, 8640, 11273, 11274, 11275}

// ruptureBaseTickDamage/ComboTickDamage are Rupture's rank -> per-tick damage
// terms, index 0 unused: the client's rank text is "(m1 + b1*cp) * ticks"
// with m1 = EffectBasePointsF and b1 = EffectPointsPerResource (1.60.1.70009
// SpellEffect.csv, spell ids 1943-11275).
var ruptureBaseTickDamage = [7]float64{0, 5, 7, 11, 16, 22, 35}
var ruptureComboTickDamage = [7]float64{0, 1.18, 1.78, 2.37, 2.96, 4.14, 4.73}

func (rogue *Rogue) registerRupture() {
	rank := core.HighestRankAtLevel(ruptureLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	spellID := ruptureSpellID[rank]

	rogue.Rupture = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueRupture,
		ActionID:      core.ActionID{SpellID: spellID},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         rogue.finisherFlags(),
		MetricSplits:  6,
		RequiredLevel: ruptureLearnLevels[rank-1],

		EnergyCost: core.EnergyCostOptions{
			Cost:   25,
			Refund: 0,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(spell.Unit.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		DamageMultiplier: []float64{1, 1.1, 1.2, 1.3}[rogue.Talents.SerratedBlades],
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rupture",
			},
			// RuptureTicks(0) (3 ticks, 6s) is this rank's registered
			// default so the conformance report's engineDuration - which
			// reads this Dot's Aura.Duration directly, before any sim runs
			// - sees the same 6000ms duration_ms the client carries for
			// every Rupture rank; ApplyEffects below always overwrites
			// this with the real per-cast combo-point value before Apply.
			NumberOfTicks: 3,
			TickLength:    time.Second * 2,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, rogue.RuptureDamage(target, rogue.ComboPoints()), isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
			if result.Landed() {
				dot := spell.Dot(target)
				dot.Spell = spell
				dot.NumberOfTicks = rogue.RuptureTicks(rogue.ComboPoints())
				dot.Apply(sim)
				rogue.SpendComboPoints(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},
	})
	rogue.Finishers = append(rogue.Finishers, rogue.Rupture)
}

func (rogue *Rogue) RuptureDamage(target *core.Unit, comboPoints int32) float64 {
	rank := core.HighestRankAtLevel(ruptureLearnLevels, rogue.Level)
	baseTickDamage := ruptureBaseTickDamage[rank]
	comboTickDamage := ruptureComboTickDamage[rank]

	return baseTickDamage + comboTickDamage*float64(comboPoints) +
		[]float64{0, 0.04 / 4, 0.10 / 5, 0.18 / 6, 0.21 / 7, 0.24 / 8}[comboPoints]*rogue.Rupture.MeleeAttackPower(target)
}

func (rogue *Rogue) RuptureTicks(comboPoints int32) int32 {
	return 3 + comboPoints
}

func (rogue *Rogue) RuptureDuration(comboPoints int32) time.Duration {
	return time.Duration(rogue.RuptureTicks(comboPoints)) * time.Second * 2
}
