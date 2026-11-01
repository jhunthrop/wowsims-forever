package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	wildGrowthTicks      = 7
	wildGrowthTickLength = time.Second
	wildGrowthCooldown   = 6 * time.Second

	// wildGrowthCastTime is every rank's live cast time. The shipped
	// client tables (SpellMisc CastingTimeIndex 1) make Wild Growth
	// instant; client 1.60.1.70707's hotfix push gives every rank a 3 sec
	// cast (the tables carry it since the site's pipeline layers the push,
	// round 6), Naturalist cuts "the cast time of your Healing Touch and
	// Wild Growth spells", and the live tooltips of all three ranks
	// (Wowhead's Forever tooltips for 408120, 1238214 and 1238215, read
	// 2026-10-31) state a 3 sec cast.
	wildGrowthCastTime = 3 * time.Second

	// wildGrowthTickDecay is the share of a tick Wild Growth moves from each
	// tick to the next: "applied quickly at first, and slows down". The client
	// states the figure as the 5 of the spell's second effect, a dummy; read
	// as a percent a tick, the first tick heals 15% over the average and the
	// last 15% under it, so the whole is still seven average ticks (the 336 of
	// the talent text is 7 x 48).
	wildGrowthTickDecay = 0.05
)

// wildGrowthTickWeight is the multiplier of the average tick for the tick-th
// tick (1-based).
func wildGrowthTickWeight(tick int32) float64 {
	middle := float64(wildGrowthTicks+1) / 2
	return 1 + wildGrowthTickDecay*(middle-float64(tick))
}

// registerWildGrowthSpell registers Wild Growth, ranks 1 to 3, when the
// druid has the talent: a 3 s cast heal over time on the target and the rest
// of its party, with a 6 s cooldown shared by its ranks.
func (druid *Druid) registerWildGrowthSpell() {
	if !druid.Talents.WildGrowth {
		return
	}
	cooldown := core.Cooldown{Timer: druid.NewTimer(), Duration: wildGrowthCooldown}

	druid.WildGrowth = druid.registerHealingRanks(wildGrowthTable, func(rank int, spec healingRank) core.SpellConfig {
		config := druid.healingSpellConfig(rank, spec, SpellCode_DruidWildGrowth, DruidSpellMaskWildGrowth)
		config.Cast.CD = cooldown
		config.Cast.DefaultCast.CastTime = wildGrowthCastTime
		config.Hot = druid.healOverTimeConfig("Wild Growth", rank, spec, wildGrowthTicks, wildGrowthTickLength)
		config.Hot.OnTick = func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			result := dot.CalcSnapshotHealing(sim, target, dot.OutcomeTick)
			result.Damage *= wildGrowthTickWeight(dot.TickCount)
			dot.Spell.DealPeriodicHealing(sim, result)
		}
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.eachPartyMember(target, func(member *core.Unit) {
				spell.Hot(member).Apply(sim)
			})
		}
		return config
	})
}
