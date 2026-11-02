package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	MarshalsPlateGauntlets = 16484
	GeneralsPlateGauntlets = 16548
	LifegivingGem          = 19341
	RageOfMugamba          = 19577
	GrileksCharmOfMight    = 19951
	DiamondFlask           = 20130
)

func init() {
	core.AddEffectsToTest = false

	// Diamond Flask: Forever's use, ItemEffect 101630, is 363881 "CHUG! CHUG!
	// CHUG! CHUG!", a 5 sec channel restoring 1120 health, then 20 Strength
	// for 1 min if it finishes (6 min cooldown), not Era's 75 Strength for 60
	// sec. A 5 sec channel in the fight costs more than 20 Strength returns,
	// and the sim runs no pre-pull channel, so it has no effect here.
	core.NewItemEffect(DiamondFlask, func(core.Agent) {})

	core.NewItemEffect(GeneralsPlateGauntlets, func(agent core.Agent) {
		warrior := agent.(WarriorAgent).GetWarrior()

		warrior.RegisterAura(core.Aura{
			Label: "Hamstring Rage Reduction",
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				warrior.Hamstring.Cost.FlatModifier -= 3
			},
		})
	})

	core.NewItemEffect(GrileksCharmOfMight, func(agent core.Agent) {
		warrior := agent.(WarriorAgent).GetWarrior()
		actionID := core.ActionID{ItemID: GrileksCharmOfMight}
		rageMetrics := warrior.NewRageMetrics(actionID)

		spell := warrior.Character.RegisterSpell(core.SpellConfig{
			ActionID:    actionID,
			SpellSchool: core.SpellSchoolPhysical,
			ProcMask:    core.ProcMaskEmpty,
			Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,

			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    warrior.NewTimer(),
					Duration: time.Minute * 3,
				},
			},

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				warrior.AddRage(sim, 30, rageMetrics)
			},
		})

		warrior.AddMajorCooldown(core.MajorCooldown{
			Type:  core.CooldownTypeDPS,
			Spell: spell,
		})
	})

	core.NewItemEffect(RageOfMugamba, func(agent core.Agent) {
		warrior := agent.(WarriorAgent).GetWarrior()

		warrior.RegisterAura(core.Aura{
			Label: "Reduces the cost of your Hamstring ability by 2 rage points.",
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				warrior.Hamstring.Cost.FlatModifier -= 2
			},
		})
	})

	core.NewItemEffect(LifegivingGem, func(agent core.Agent) {
		warrior := agent.(WarriorAgent).GetWarrior()
		actionID := core.ActionID{ItemID: LifegivingGem}
		healthMetrics := warrior.NewHealthMetrics(actionID)

		var bonusHealth float64
		lifegivingGemAura := warrior.RegisterAura(core.Aura{
			Label:    "Gift of Life",
			ActionID: core.ActionID{SpellID: 23725},
			Duration: time.Second * 20,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				bonusHealth = warrior.MaxHealth() * 0.15
				warrior.AddStatsDynamic(sim, stats.Stats{stats.Health: bonusHealth})
				warrior.GainHealth(sim, bonusHealth, healthMetrics)
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				warrior.AddStatsDynamic(sim, stats.Stats{stats.Health: -bonusHealth})
			},
		})

		lifegivingGemSpell := warrior.RegisterSpell(AnyStance, core.SpellConfig{
			ActionID: actionID,

			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    warrior.NewTimer(),
					Duration: time.Minute * 5,
				},
			},

			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
				lifegivingGemAura.Activate(sim)
			},
		})

		warrior.AddMajorCooldown(core.MajorCooldown{
			Spell: lifegivingGemSpell.Spell,
			Type:  core.CooldownTypeSurvival,
		})
	})

	core.NewItemEffect(MarshalsPlateGauntlets, func(agent core.Agent) {
		warrior := agent.(WarriorAgent).GetWarrior()

		warrior.RegisterAura(core.Aura{
			Label: "Hamstring Rage Reduction",
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				warrior.Hamstring.Cost.FlatModifier -= 3
			},
		})
	})

	core.AddEffectsToTest = true
}
