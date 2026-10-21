package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	bloodrageInstantRage   = 10.0
	bloodrageRagePerSecond = 1.0
)

func (warrior *Warrior) registerBloodrageCD() {
	actionID := core.ActionID{SpellID: 2687}
	rageMetrics := warrior.NewRageMetrics(actionID)

	// Bloodrage (client spell 2687 and its periodic 29131): an instant
	// 10 rage (effect amount 100, tenths) and 1 rage (amount 10) a second
	// for 10 seconds. Improved Bloodrage scales both.
	rageScale := warrior.improvedBloodrageMultiplier()
	instantRage := bloodrageInstantRage * rageScale
	ragePerSec := bloodrageRagePerSecond * rageScale

	warrior.BloodrageAura = warrior.RegisterAura(core.Aura{
		Label:    "Bloodrage",
		ActionID: actionID,
		Duration: time.Second * 10,
	})

	warrior.Bloodrage = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID: actionID,
		// The Forever Fury rotation casts Bloodrage by id rather than
		// leaving it to the cooldown autocaster, so it has to be a spell
		// the APL can name.
		Flags: core.SpellFlagAPL,

		RequiredLevel: BloodrageLevel[0],
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: time.Minute,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			warrior.BloodrageAura.Activate(sim)
			warrior.AddRage(sim, instantRage, rageMetrics)

			core.StartPeriodicAction(sim, core.PeriodicActionOptions{
				NumTicks: 10,
				Period:   time.Second * 1,
				OnAction: func(sim *core.Simulation) {
					warrior.AddRage(sim, ragePerSec, rageMetrics)
				},
			})
		},
	})

	warrior.AddMajorCooldown(core.MajorCooldown{
		Spell: warrior.Bloodrage.Spell,
		Type:  core.CooldownTypeDPS,
	})
}
