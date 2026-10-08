package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Enrage (spell 5229, learned at 12; build 1.60.1.70009): a Bear Form
// ability with no cost or global cooldown and a 60 second cooldown. The
// client gives it an instant energize of 10 rage (effect 30, amount 100 in
// tenths of a point) and a periodic energize aura (aura 24, 20 every second,
// 2 rage) for 10 seconds, so a cast pays 30 rage; the bear also "loses" base
// armor for the duration, in a dummy effect with no amount.
//
// unconfirmed: the armor penalty. Era's 27% of base armor is used, since the
// client's dummy effect states none. The instant 10 rage and the 2 a second
// are the client's amounts read as tenths of a point, the unit rage costs
// use.
const (
	enrageSpellID            = 5229
	enrageDuration           = 10 * time.Second
	enrageCooldown           = time.Minute
	enrageLearnLevel         = 12
	enrageInstantRage        = 10.0
	enrageRagePerTick        = 2.0
	enrageTickPeriod         = time.Second
	enrageBaseArmorReduction = 0.27
)

func (druid *Druid) registerEnrageSpell() {
	if int(druid.Level) < enrageLearnLevel {
		return
	}
	actionID := core.ActionID{SpellID: enrageSpellID}
	rageMetrics := druid.NewRageMetrics(actionID)
	armorMultiplier := 1 - enrageBaseArmorReduction

	druid.EnrageAura = druid.RegisterAura(core.Aura{
		Label:    "Enrage Aura",
		ActionID: actionID,
		Duration: enrageDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, pool := range bearFormArmorPools {
				druid.ApplyDynamicEquipScaling(sim, pool, armorMultiplier)
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, pool := range bearFormArmorPools {
				druid.RemoveDynamicEquipScaling(sim, pool, armorMultiplier)
			}
		},
	})

	druid.Enrage = druid.RegisterSpell(Bear, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagAPL,

		RequiredLevel: enrageLearnLevel,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: enrageCooldown,
			},
			IgnoreHaste: true,
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			druid.AddRage(sim, enrageInstantRage, rageMetrics)

			core.StartPeriodicAction(sim, core.PeriodicActionOptions{
				NumTicks: int(enrageDuration / enrageTickPeriod),
				Period:   enrageTickPeriod,
				OnAction: func(sim *core.Simulation) {
					if druid.EnrageAura.IsActive() {
						druid.AddRage(sim, enrageRagePerTick, rageMetrics)
					}
				},
			})

			druid.EnrageAura.Activate(sim)
		},
	})
}
