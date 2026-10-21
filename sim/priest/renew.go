package priest

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	// renewTicks is the client's 15 s duration at one tick per 3 s;
	// TestRenewTicksFollowTheClientDuration pins both.
	renewTicks      = 5
	renewTickLength = 3 * time.Second
)

// registerRenew registers every rank of Renew. The client's amount is per
// tick and the coefficient is each tick's share of spell power.
func (priest *Priest) registerRenew() {
	priest.Renew = priest.registerHealRanks(renewRanks, func(rank int, entry healRank) core.SpellConfig {
		config := priest.healSpellConfig(entry, rank, SpellCode_PriestRenew, PriestSpellMaskRenew)
		tick := entry.effect.Center(int(priest.Level))
		config.Hot = core.DotConfig{
			Aura: core.Aura{
				Label: fmt.Sprintf("Renew (Rank %d)", rank),
			},
			NumberOfTicks:    renewTicks,
			TickLength:       renewTickLength,
			BonusCoefficient: entry.coefficient,

			OnSnapshot: func(_ *core.Simulation, target *core.Unit, dot *core.Dot, _ bool) {
				snapshotHeal(dot, target, tick)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotHealing(sim, target, dot.Spell.OutcomeHealing)
			},
		}
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Hot(target).Apply(sim)
		}
		return config
	})
}

// snapshotHeal takes a heal-over-time's per-tick amount: the base tick
// plus the healing stat times the tick's coefficient, and the caster's
// healing multipliers. core.Dot.SnapshotHeal reads the damage multipliers
// instead, which would leave out Spiritual Healing and Power Infusion.
func snapshotHeal(dot *core.Dot, target *core.Unit, baseTick float64) {
	dot.SnapshotBaseDamage = baseTick + dot.BonusCoefficient*dot.Spell.HealingPower(target)
	dot.SnapshotAttackerMultiplier = dot.Spell.CasterHealingMultiplier()
}
