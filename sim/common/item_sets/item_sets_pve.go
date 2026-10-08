package item_sets

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

///////////////////////////////////////////////////////////////////////////
//                                 Other
///////////////////////////////////////////////////////////////////////////

var ItemSetSpidersKiss = core.NewItemSet(core.ItemSet{
	Name: "Spider's Kiss",
	Bonuses: map[int32]core.ApplyEffect{
		// Chance on Hit: Immobilizes the target and lowers their armor by 100 for 10 sec.
		// Unsure about exlusive effects with this aura also looks like it might be lowering the characters armor instead of the enemy?
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			procAura := character.NewTemporaryStatsAura("Spider's Kiss", core.ActionID{SpellID: 17333}, stats.Stats{stats.Armor: -100}, time.Second*10)
			core.MakeProcTriggerAura(&character.Unit, core.ProcTrigger{
				ActionID:   core.ActionID{SpellID: 17333},
				Name:       "Spider's Kiss",
				Callback:   core.CallbackOnSpellHitDealt,
				Outcome:    core.OutcomeLanded,
				ProcMask:   core.ProcMaskMelee,
				ProcChance: 0.05,
				Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
					procAura.Activate(sim)
				},
			})
		},
	},
})

var ItemSetDalRendsArms = core.NewItemSet(core.ItemSet{
	Name: "Dal'Rend's Arms",
	Bonuses: map[int32]core.ApplyEffect{
		// +50 Attack Power.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStat(stats.AttackPower, 50)
			character.AddStat(stats.RangedAttackPower, 50)
		},
	},
})

var ItemSetShardOfTheGods = core.NewItemSet(core.ItemSet{
	Name: "Shard of the Gods",
	Bonuses: map[int32]core.ApplyEffect{
		// +10 All Resistances.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddResistances(15)
		},
	},
})

var ItemSetMajorMojoInfusion = core.NewItemSet(core.ItemSet{
	Name: "Major Mojo Infusion",
	Bonuses: map[int32]core.ApplyEffect{
		// +30 Attack Power.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStats(stats.Stats{
				stats.AttackPower:       30,
				stats.RangedAttackPower: 30,
			})
		},
	},
})

var ItemSetOverlordsResolution = core.NewItemSet(core.ItemSet{
	Name: "Overlord's Resolution",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases our chance to dodge an attack by 1%
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStat(stats.Dodge, 1)
		},
	},
})

var ItemSetPrayerOfThePrimal = core.NewItemSet(core.ItemSet{
	Name: "Prayer of the Primal",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases healing done by up to 33
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStat(stats.HealingPower, 33)
		},
	},
})

var ItemSetPrimalBlessing = core.NewItemSet(core.ItemSet{
	Name: "Primal Blessing",
	Bonuses: map[int32]core.ApplyEffect{
		// Grants a small chance when ranged or melee damage is dealt to infuse the wielder with a blessing from the Primal Gods.
		// Ranged and melee attack power increased by 300 for 12 sec.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()

			aura := character.RegisterAura(core.Aura{
				ActionID: core.ActionID{SpellID: 467742},
				Label:    "Primal Blessing",
				Duration: time.Second * 12,
				OnGain: func(aura *core.Aura, sim *core.Simulation) {
					character.AddStatsDynamic(sim, stats.Stats{
						stats.AttackPower:       300,
						stats.RangedAttackPower: 300,
					})
				},
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					character.AddStatsDynamic(sim, stats.Stats{
						stats.AttackPower:       -300,
						stats.RangedAttackPower: -300,
					})
				},
			})

			core.MakeProcTriggerAura(&character.Unit, core.ProcTrigger{
				Name:       "Primal Blessing Trigger",
				Callback:   core.CallbackOnSpellHitDealt,
				ProcMask:   core.ProcMaskMeleeOrRanged,
				Outcome:    core.OutcomeLanded,
				ProcChance: 0.05,
				ICD:        time.Second * 72,
				Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					aura.Activate(sim)
				},
			})
		},
	},
})

var ItemSetTwinBladesofHakkari = core.NewItemSet(core.ItemSet{
	Name: "The Twin Blades of Hakkari",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases Swords +6
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.PseudoStats.SwordsSkill += 6
		},
	},
})

var ItemSetZanzilsConcentration = core.NewItemSet(core.ItemSet{
	Name: "Zanzil's Concentration",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases damage and healing done by magical spells and effects by up to 6.
		// Improves your chance to hit with all spells and attacks by 1%.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStats(stats.Stats{
				stats.SpellPower: 6,
				stats.Hit:        1 * core.HitRatingPerHitChance,
			})
		},
	},
})
