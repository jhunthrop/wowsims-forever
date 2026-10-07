package priest

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const DevouringPlagueRanks = 6

var DevouringPlagueSpellId = [DevouringPlagueRanks + 1]int32{0, 2944, 19276, 19277, 19278, 19279, 19280}
var DevouringPlagueBaseDamage = [DevouringPlagueRanks + 1]float64{0, 152, 272, 400, 544, 712, 904}
var DevouringPlagueManaCost = [DevouringPlagueRanks + 1]float64{0, 215, 350, 495, 645, 810, 985}
var DevouringPlagueLevel = [DevouringPlagueRanks + 1]int{0, 20, 28, 36, 44, 52, 60}

func (priest *Priest) registerDevouringPlagueSpell() {
	//TO DO: Implement race requirement
	priest.DevouringPlague = make([]*core.Spell, DevouringPlagueRanks+1)
	cdTimer := priest.NewTimer()

	for rank := 1; rank <= DevouringPlagueRanks; rank++ {
		config := priest.getDevouringPlagueConfig(rank, cdTimer)

		if config.RequiredLevel <= int(priest.Level) {
			priest.DevouringPlague[rank] = priest.GetOrRegisterSpell(config)
		}
	}
}

func (priest *Priest) getDevouringPlagueConfig(rank int, cdTimer *core.Timer) core.SpellConfig {

	var ticks int32 = 8

	spellId := DevouringPlagueSpellId[rank]
	baseDotDamage := (DevouringPlagueBaseDamage[rank] / float64(ticks))
	manaCost := DevouringPlagueManaCost[rank]
	level := DevouringPlagueLevel[rank]

	// The Forever client's spellconst (1.60.1.70009, priest.json spells
	// 2944/19276/19277/19278/19279/19280, effect 0's sp_coefficient) is
	// a flat 0.1 on every one of Devouring Plague's six ranks, not
	// 0.063 - verified directly against the build's own data. Rotation-
	// accuracy program, 2026-09-28 (audit-priest).
	spellCoeff := 0.1

	return core.SpellConfig{
		SpellCode:      SpellCode_PriestDevouringPlague,
		ActionID:       core.ActionID{SpellID: spellId},
		SpellSchool:    core.SpellSchoolShadow,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          SpellFlagPriest | core.SpellFlagAPL | core.SpellFlagDisease | core.SpellFlagPureDot,
		ClassSpellMask: PriestSpellMaskDevouringPlague,

		Rank:          rank,
		RequiredLevel: level,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer: cdTimer,
				// The client's category_cooldown_ms is 60000 (1 min) for
				// every rank (spellconst/priest.json, build 1.60.1.70009),
				// not 3 min - conformance golden
				// sim/core/testdata/conformance/priest.golden.md flagged
				// this as cooldown_ms 60000->180000 (client->engine).
				Duration: time.Minute,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: fmt.Sprintf("Devouring Plague (Rank %d)", rank),
			},

			NumberOfTicks:    ticks,
			TickLength:       time.Second * 3,
			BonusCoefficient: spellCoeff,
			// Blizzard's 1 October 2026 notes: "Devouring Plague can
			// crit". Each tick rolls the priest's spell crit.
			CanCrit: true,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDotDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeMagicCritPerTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				priest.AddShadowWeavingStack(sim, target)
				spell.Dot(target).Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},
	}
}
