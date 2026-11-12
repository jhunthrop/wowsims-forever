package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	bloodrageInstantRage   = 10.0
	bloodrageRagePerSecond = 1.0
	// bloodrageHealthCostShare is the client's cost: SpellPower row for
	// 2687 has PowerType -2 (health) and PowerCostPct 20, a fifth of the
	// warrior's base health (the class's own health at the level, before
	// Stamina: 1,689 at 60, so 338 health a cast). Vanilla's Bloodrage
	// cost 16%; the engine had charged nothing.
	bloodrageHealthCostShare = 0.20
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

	healthCost := bloodrageHealthCostShare * warrior.GetBaseStats()[stats.Health]

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

		ExtraCastCondition: func(sim *core.Simulation, _ *core.Unit) bool {
			// A health cost needs the health: the client refuses the
			// cast below it rather than letting it kill the warrior.
			return warrior.CurrentHealth() > healthCost
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			warrior.RemoveHealth(sim, healthCost)
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
