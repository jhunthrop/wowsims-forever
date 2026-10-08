package priest

import (
	"github.com/wowsims/classic/sim/core"
)

// registerHolyNova registers Holy Nova, which exists only with its talent.
// The cast spell carries the damage half the client states on it and the
// heal half is a helper spell (holyNovaHealRanks); both have the same
// 0.107 coefficient, so the spell's own coefficient serves both. It heals
// the caster's party and damages every enemy.
func (priest *Priest) registerHolyNova() {
	if !priest.Talents.HolyNova {
		return
	}
	priest.HolyNova = priest.registerHealRanks(holyNovaDamageRanks, func(rank int, entry healRank) core.SpellConfig {
		heal := holyNovaHealRanks[rank-1]
		config := priest.healSpellConfig(entry, rank, SpellCode_PriestHolyNova, PriestSpellMaskHolyNova)
		config.ProcMask = core.ProcMaskSpellDamage | core.ProcMaskSpellHealing
		config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			for _, member := range partyOf(&priest.Unit) {
				priest.healTarget(sim, spell, member, priest.roll(sim, heal))
			}
			for _, enemy := range sim.Encounter.TargetUnits {
				spell.CalcAndDealDamage(sim, enemy, priest.roll(sim, entry), spell.OutcomeMagicHitAndCrit)
			}
		}
		return config
	})
}
