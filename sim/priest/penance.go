package priest

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	penanceCooldown = 12 * time.Second
	// penanceChannelTicks are the pulses after the instant first one:
	// "instantly and every 1 sec for 2 sec".
	penanceChannelTicks = 2
	penanceTickLength   = time.Second
)

// registerPenance registers Penance's heal. The client splits the volley
// across helper spells; the cast spell carries the cost and cooldown and
// the heal helper (healRank.effectSpellID) the amount, so one channel
// here lands the instant pulse and two more on the second boundaries. Its
// damage half, thrown at an enemy, is not modelled: no healing rotation
// casts it.
func (priest *Priest) registerPenance() {
	if !priest.Talents.Penance {
		return
	}
	cooldown := core.Cooldown{Timer: priest.NewTimer(), Duration: penanceCooldown}
	priest.Penance = priest.registerHealRanks(penanceRanks, func(rank int, entry healRank) core.SpellConfig {
		config := priest.healSpellConfig(entry, rank, SpellCode_PriestPenance, PriestSpellMaskPenance)
		config.Flags |= core.SpellFlagChanneled
		config.Cast.CD = cooldown
		config.Hot = core.DotConfig{
			Aura: core.Aura{
				Label: fmt.Sprintf("Penance (Rank %d)", rank),
			},
			NumberOfTicks: penanceChannelTicks,
			TickLength:    penanceTickLength,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				priest.healTarget(sim, dot.Spell, target, priest.roll(sim, entry))
			},
		}
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			priest.healTarget(sim, spell, target, priest.roll(sim, entry))
			spell.Hot(target).Apply(sim)
		}
		return config
	})
}
