package druid

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// HurricaneChannelSpellId is the channel the druid casts (16914, 17401,
// 17402). Its own effect 0 is a dummy; its area trigger casts the tick
// spell every second, and the generated Hurricane* arrays
// (constants_auto_gen.go) are those tick spells' rows: HurricaneSpellId
// 1278965, 1278968 and 1278759 (rank 0 is a creature variant), with the
// amount a tick does, its growth per level and its cap. A hand table once
// read 70, 100 and 134 for the client's 68, 98 and 132 (2026-11-11
// casters lane); sim/druid/spellconst_damage_test.go now holds every rank
// to the vendored client file.
var HurricaneChannelSpellId = [HurricaneRanks + 1]int32{0, 16914, 17401, 17402}

// HurricaneChannelManaCost is the channel's cost (spellconst 16914, 17401,
// 17402); the tick spells cost nothing, so the generated HurricaneManaCost
// reads 0 for them.
var HurricaneChannelManaCost = [HurricaneRanks + 1]float64{0, 880, 1180, 1495}

// HurricaneTickDamage is each rank's tick as a client damage effect, from
// the generated rows.
var HurricaneTickDamage = clientdamage.FromTable(HurricaneBaseDamage[:], HurricanePointsPerLevel[:], HurricaneLevel[:], HurricaneMaxLevel[:])

func (druid *Druid) registerHurricaneSpell() {
	for rank := 1; rank <= HurricaneRanks; rank++ {
		level := HurricaneLevel[rank]
		if int(druid.Level) < level {
			break
		}

		tickDamage := HurricaneTickDamage[rank]
		casterLevel := int(druid.Level)
		spell := druid.RegisterSpell(Humanoid|Moonkin, core.SpellConfig{
			ActionID:    core.ActionID{SpellID: HurricaneChannelSpellId[rank]},
			SpellSchool: core.SpellSchoolNature,
			ProcMask:    core.ProcMaskSpellDamage,
			Flags:       SpellFlagOmen | core.SpellFlagChanneled | core.SpellFlagBinary | core.SpellFlagAPL,

			RequiredLevel: level,
			Rank:          rank,

			ManaCost: core.ManaCostOptions{
				FlatCost: HurricaneChannelManaCost[rank],
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
			ClientBaseDamage: tickDamage.Range(casterLevel),

			Dot: core.DotConfig{
				IsAOE: true,
				Aura: core.Aura{
					Label: "Hurricane" + druid.Label + strconv.Itoa(rank),
				},
				// Each pulse is the server tick (1278759, 1278965, 1278968): direct school damage with no
				// SPELL_ATTR2_CANT_CRIT, so it crits like any direct spell; the
				// channel itself is a dummy aura with no periodic effect for
				// PERIODIC_CAN_CRIT to mark (crit-flags sweep, 2026-11-13).
				CanCrit:       true,
				NumberOfTicks: 10,
				TickLength:    time.Second * 1,

				BonusCoefficient: HurricaneSpellCoeff[rank],
				Dynamic:          true,

				OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
					dot.Snapshot(target, tickDamage.Center(casterLevel), isRollover)
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
