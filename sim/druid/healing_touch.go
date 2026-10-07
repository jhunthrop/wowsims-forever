package druid

import "github.com/wowsims/classic/sim/core"

// registerHealingTouchSpell registers Healing Touch, ranks 1 to 10 (11 is
// the Ahn'Qiraj book rank). Its cast time, mana cost and coefficient are the
// client's own: ranks 1 to 4 are quicker casts with smaller coefficients,
// from rank 5 on it is a 3.5 s cast at coefficient 1.
func (druid *Druid) registerHealingTouchSpell() {
	table := healingTouchTable[:core.MaxTrainerRank(len(healingTouchTable))]
	druid.HealingTouch = druid.registerHealingRanks(table, func(rank int, spec healingRank) core.SpellConfig {
		config := druid.healingSpellConfig(rank, spec, SpellCode_DruidHealingTouch, DruidSpellMaskHealingTouch)
		config.ApplyEffects = druid.directHealEffects(spec)
		return config
	})
}
