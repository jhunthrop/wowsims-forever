package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	rejuvenationTicks      = 4
	rejuvenationTickLength = 3 * time.Second // 12 s in all
)

// registerRejuvenationSpell registers Rejuvenation, ranks 1 to 10 (11 is the
// Ahn'Qiraj book rank): an instant heal over time.
func (druid *Druid) registerRejuvenationSpell() {
	table := rejuvenationTable[:core.MaxTrainerRank(len(rejuvenationTable))]
	druid.Rejuvenation = druid.registerHealingRanks(table, func(rank int, spec healingRank) core.SpellConfig {
		config := druid.healingSpellConfig(rank, spec, SpellCode_DruidRejuvenation, DruidSpellMaskRejuvenation)
		config.Hot = druid.healOverTimeConfig("Rejuvenation", rank, spec, rejuvenationTicks, rejuvenationTickLength)
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Hot(target).Apply(sim)
		}
		return config
	})
}
