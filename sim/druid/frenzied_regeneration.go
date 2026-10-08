package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Frenzied Regeneration (spell 22842, learned at 36; build 1.60.1.70009): a
// Bear Form ability on the global cooldown with a 3 minute cooldown that
// lasts 10 seconds. The client row holds only the periodic aura (aura 23,
// amount 1, every second); the conversion is the Era spell's: each second it
// spends up to 10 rage and heals 0.3% of maximum health for every point of
// rage spent.
//
// unconfirmed: the whole conversion, because the client states none of it.
const (
	frenziedRegenerationSpellID       = 22842
	frenziedRegenerationDuration      = 10 * time.Second
	frenziedRegenerationCooldown      = 3 * time.Minute
	frenziedRegenerationLearnLevel    = 36
	frenziedRegenerationRagePerSecond = 10.0
	frenziedRegenerationHealthPerRage = 0.003
	frenziedRegenerationTickPeriod    = time.Second
)

func (druid *Druid) registerFrenziedRegenerationCD() {
	if int(druid.Level) < frenziedRegenerationLearnLevel {
		return
	}
	actionID := core.ActionID{SpellID: frenziedRegenerationSpellID}
	healthMetrics := druid.NewHealthMetrics(actionID)
	rageMetrics := druid.NewRageMetrics(actionID)

	druid.FrenziedRegenerationAura = druid.RegisterAura(core.Aura{
		Label:    "Frenzied Regeneration",
		ActionID: actionID,
		Duration: frenziedRegenerationDuration,
	})

	druid.FrenziedRegeneration = druid.RegisterSpell(Bear, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagAPL,

		RequiredLevel: frenziedRegenerationLearnLevel,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: frenziedRegenerationCooldown,
			},
			IgnoreHaste: true,
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			core.StartPeriodicAction(sim, core.PeriodicActionOptions{
				NumTicks: int(frenziedRegenerationDuration / frenziedRegenerationTickPeriod),
				Period:   frenziedRegenerationTickPeriod,
				OnAction: func(sim *core.Simulation) {
					if !druid.FrenziedRegenerationAura.IsActive() {
						return
					}
					rageSpent := min(druid.CurrentRage(), frenziedRegenerationRagePerSecond)
					healthGained := rageSpent * frenziedRegenerationHealthPerRage * druid.MaxHealth() * druid.PseudoStats.HealingTakenMultiplier
					druid.SpendRage(sim, rageSpent, rageMetrics)
					druid.GainHealth(sim, healthGained, healthMetrics)
				},
			})

			druid.FrenziedRegenerationAura.Activate(sim)
		},
	})

	druid.AddMajorCooldown(core.MajorCooldown{
		Spell: druid.FrenziedRegeneration.Spell,
		Type:  core.CooldownTypeSurvival,
	})
}
