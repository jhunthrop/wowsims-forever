package druid

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

var ItemSetCenarionRaiment = core.NewItemSet(core.ItemSet{
	Name: "Cenarion Raiment",
	Bonuses: map[int32]core.ApplyEffect{
		// (3) Set : Damage dealt by Thorns increased by 4 and duration increased by 50%.
		3: func(agent core.Agent) {
			// Nothing to do
		},
		// (5) Set : Improves your chance to get a critical strike with spells by 2%.
		5: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.Crit, 2*core.CritRatingPerCritChance)
		},
		// (8) Set : Reduces the cooldown of your Tranquility and Hurricane spells by 50%.
		8: func(agent core.Agent) {
			// Nothing to do
		},
	},
})

var ItemSetStormrageRaiment = core.NewItemSet(core.ItemSet{
	Name: "Stormrage Raiment",
	Bonuses: map[int32]core.ApplyEffect{
		// (3) Set : Allows 15% of your Mana regeneration to continue while casting.
		3: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.PseudoStats.SpiritRegenRateCasting += .15
		},
		// (5) Set : Reduces the casting time of your Regrowth spell by 0.2 sec.
		5: func(agent core.Agent) {
			// Nothing to do.
		},
		// (8) Set : Increases the duration of your Rejuvenation spell by 3 sec.
		8: func(agent core.Agent) {
			// Nothing to do.
		},
	},
})

var ItemSetSymbolsOfUnendingLife = core.NewItemSet(core.ItemSet{
	Name: "Symbols of Unending Life",
	Bonuses: map[int32]core.ApplyEffect{
		// (3) Set : Your finishing moves now refund 30 energy on a Miss, Dodge, Block, or Parry.
		3: func(agent core.Agent) {
			c := agent.GetCharacter()
			actionID := core.ActionID{SpellID: 26107}
			energyMetrics := c.NewEnergyMetrics(actionID)
			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:     "Symbols of Unending Life Finisher Bonus",
				Callback: core.CallbackOnSpellHitDealt,
				Outcome:  core.OutcomeMiss | core.OutcomeDodge | core.OutcomeBlock | core.OutcomeParry,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					if spell.SpellCode == SpellCode_DruidFerociousBite || spell.SpellCode == SpellCode_DruidRip {
						c.AddEnergy(sim, 30, energyMetrics)
					}
				},
			})
		},
	},
})
