package priest

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

const StarshardsTicks = 6

func (priest *Priest) registerStarshardsSpell() {
	if priest.Race != proto.Race_RaceNightElf {
		return
	}

	priest.Starshards = make([][]*core.Spell, StarshardsRanks+1)

	for rank := 1; rank <= StarshardsRanks; rank++ {
		priest.Starshards[rank] = make([]*core.Spell, StarshardsTicks+1)

		var tick int32
		for tick = 0; tick < StarshardsTicks; tick++ {
			config := priest.newStarshardsSpellConfig(rank, tick)

			if config.RequiredLevel <= int(priest.Level) {
				priest.Starshards[rank][tick] = priest.RegisterSpell(config)
			}
		}
	}
}

func (priest *Priest) newStarshardsSpellConfig(rank int, tickIdx int32) core.SpellConfig {
	ticks := tickIdx
	flags := SpellFlagPriest | core.SpellFlagChanneled | core.SpellFlagBinary
	if tickIdx == 0 {
		ticks = 6
		flags |= core.SpellFlagAPL
	}

	spellId := StarshardsSpellId[rank]
	roll := priest.clientRoll(StarshardsBaseDamage[rank], StarshardsPointsPerLevel[rank], StarshardsLevel[rank], StarshardsMaxLevel[rank])
	baseDamage := periodicTick(roll) // per tick, whatever the channel's length
	manaCost := StarshardsManaCost[rank]
	level := StarshardsLevel[rank]

	spellCoeff := StarshardsSpellCoeff[rank]

	tickLength := time.Second

	return core.SpellConfig{
		SpellCode:   SpellCode_PriestStarshards,
		ActionID:    core.ActionID{SpellID: spellId}.WithTag(tickIdx),
		SpellSchool: core.SpellSchoolArcane,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       flags,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: roll,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: fmt.Sprintf("Starshards-%d-%d", rank, tickIdx),
			},
			NumberOfTicks:       ticks,
			TickLength:          tickLength,
			AffectedByCastSpeed: false,
			BonusCoefficient:    spellCoeff,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				spell.Dot(target).Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},

		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			result := spell.CalcPeriodicDamage(sim, target, baseDamage, spell.OutcomeExpectedMagicAlwaysHit)
			return result
		},
	}
}
