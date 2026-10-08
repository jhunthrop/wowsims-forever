package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	regrowthTicks      = 7
	regrowthTickLength = 3 * time.Second // 21 s in all
)

// registerRegrowthSpell registers Regrowth, ranks 1 to 9: a 2 s cast that
// heals at once and leaves a heal over time on the target. Only the direct
// heal can critically strike (Improved Regrowth).
func (druid *Druid) registerRegrowthSpell() {
	druid.Regrowth = druid.registerHealingRanks(regrowthTable, func(rank int, spec healingRank) core.SpellConfig {
		config := druid.healingSpellConfig(rank, spec, SpellCode_DruidRegrowth, DruidSpellMaskRegrowth)
		config.Hot = druid.healOverTimeConfig("Regrowth", rank, spec, regrowthTicks, regrowthTickLength)

		directHeal := druid.directHealEffects(spec)
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			directHeal(sim, target, spell)
			spell.Hot(target).Apply(sim)
		}
		return config
	})
}
