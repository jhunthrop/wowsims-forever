package druid

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (druid *Druid) registerHurricaneSpell() {
	ranks := []struct {
		level      int32
		spellID    int32
		manaCost   float64
		scaleLevel int32
		damage     float64
		scale      float64
	}{
		{level: 40, spellID: 16914, manaCost: 880, scaleLevel: 46, damage: 70, scale: 0.2},
		{level: 50, spellID: 17401, manaCost: 1180, scaleLevel: 56, damage: 100, scale: 0.2},
		{level: 60, spellID: 17402, manaCost: 1495, scaleLevel: 66, damage: 134, scale: 0.3},
	}

	for i, rank := range ranks {
		if druid.Level < rank.level {
			break
		}

		damage := rank.damage + float64(min(druid.Level, rank.scaleLevel)-rank.level)*rank.scale
		spell := druid.RegisterSpell(Humanoid|Moonkin, core.SpellConfig{
			ActionID:    core.ActionID{SpellID: rank.spellID},
			SpellSchool: core.SpellSchoolNature,
			ProcMask:    core.ProcMaskSpellDamage,
			Flags:       SpellFlagOmen | core.SpellFlagChanneled | core.SpellFlagBinary | core.SpellFlagAPL,

			RequiredLevel: int(rank.level),
			Rank:          i + 1,

			ManaCost: core.ManaCostOptions{
				FlatCost: rank.manaCost,
			},
			// Hurricane carries no cooldown in 1.60.1.70009 spellconst/
			// druid.json (cooldown_ms and category_cooldown_ms are both
			// 0 on every id for this name); the 60s CD below was a
			// stale pre-Forever value and has been removed.
			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD: core.GCDDefault,
				},
			},

			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			Dot: core.DotConfig{
				IsAOE: true,
				Aura: core.Aura{
					Label: "Hurricane" + druid.Label + strconv.Itoa(i+1),
				},
				NumberOfTicks: 10,
				TickLength:    time.Second * 1,

				BonusCoefficient: 0.03,

				OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
					dot.Snapshot(target, damage, isRollover)
				},
				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					for _, aoeTarget := range sim.Encounter.TargetUnits {
						dot.CalcAndDealPeriodicSnapshotDamage(sim, aoeTarget, dot.OutcomeTick)
					}
				},
			},

			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
				druid.AutoAttacks.CancelAutoSwing(sim)
				spell.AOEDot().Apply(sim)
			},
		})

		druid.Hurricane = append(druid.Hurricane, spell)
	}
}
