package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	tranquilityTicks      = 5
	tranquilityTickLength = 2 * time.Second // a 10 s channel
	tranquilityCooldown   = 5 * time.Minute
)

// registerTranquilitySpell registers Tranquility, ranks 1 to 4: a channel
// that heals the druid's whole party every 2 s. The client states the
// effect as an area aura on the party (effect 35), so each tick heals every
// member of the druid's party, the druid included. The channel is a heal over
// time on the druid himself (not a self-only dot, which the periodic-bonus
// spell mods, Genesis among them, skip) whose ticks go out to the party.
func (druid *Druid) registerTranquilitySpell() {
	cooldown := core.Cooldown{Timer: druid.NewTimer(), Duration: tranquilityCooldown}

	druid.Tranquility = druid.registerHealingRanks(tranquilityTable, func(rank int, spec healingRank) core.SpellConfig {
		config := druid.healingSpellConfig(rank, spec, SpellCode_DruidTranquility, DruidSpellMaskTranquility)
		config.Flags |= core.SpellFlagChanneled
		config.Cast.CD = cooldown
		config.Hot = druid.healOverTimeConfig("Tranquility", rank, spec, tranquilityTicks, tranquilityTickLength)
		config.Hot.OnTick = func(sim *core.Simulation, _ *core.Unit, dot *core.Dot) {
			druid.eachPartyMember(&druid.Unit, func(member *core.Unit) {
				dot.Spell.DealPeriodicHealing(sim, dot.CalcSnapshotHealing(sim, member, dot.OutcomeTick))
			})
		}
		config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			spell.Hot(&druid.Unit).Apply(sim)
		}
		return config
	})
}
