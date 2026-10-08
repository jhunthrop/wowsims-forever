package priest

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	// powerWordShieldCooldown is the client's category cooldown, shared by
	// every rank; Soul Warding removes it.
	powerWordShieldCooldown = 4 * time.Second
	powerWordShieldDuration = 30 * time.Second

	weakenedSoulSpellID  = 6788
	weakenedSoulDuration = 15 * time.Second
)

// registerWeakenedSoul gives every friendly unit the Weakened Soul debuff
// that keeps a second Power Word: Shield off it.
func (priest *Priest) registerWeakenedSoul() {
	priest.WeakenedSouls = priest.NewRaidAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    "Weakened Soul",
			ActionID: core.ActionID{SpellID: weakenedSoulSpellID},
			Duration: weakenedSoulDuration,
		})
	})
}

// weakenedSoul is the debuff on target, nil when target cannot carry one
// (an enemy).
func (priest *Priest) weakenedSoul(target *core.Unit) *core.Aura {
	if int(target.UnitIndex) >= len(priest.WeakenedSouls) {
		return nil
	}
	return priest.WeakenedSouls.Get(target)
}

// registerPowerWordShield registers every rank of Power Word: Shield. The
// absorb is the client's amount plus 10% of the healing stat; Improved
// Power Word: Shield scales it through the spell's damage multiplier,
// which core.Shield.Apply reads.
func (priest *Priest) registerPowerWordShield() {
	cooldown := core.Cooldown{Timer: priest.NewTimer(), Duration: powerWordShieldCooldown}
	priest.PowerWordShield = priest.registerHealRanks(powerWordShieldRanks, func(rank int, entry healRank) core.SpellConfig {
		config := priest.healSpellConfig(entry, rank, SpellCode_PriestPowerWordShield, PriestSpellMaskPowerWordShield)
		config.Cast.CD = cooldown
		config.Shield = core.ShieldConfig{
			Aura: core.Aura{
				Label:    fmt.Sprintf("Power Word: Shield (Rank %d)", rank),
				Duration: powerWordShieldDuration,
			},
		}
		config.ExtraCastCondition = func(_ *core.Simulation, target *core.Unit) bool {
			weakened := priest.weakenedSoul(target)
			return weakened != nil && !weakened.IsActive()
		}
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			absorb := entry.effect.Center(int(priest.Level)) + entry.coefficient*spell.HealingPower(target)
			spell.Shield(target).Apply(sim, absorb)
			priest.weakenedSoul(target).Activate(sim)
		}
		return config
	})
}
