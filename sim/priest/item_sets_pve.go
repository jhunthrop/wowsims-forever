package priest

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 1 Item Sets - Molten Core
///////////////////////////////////////////////////////////////////////////

var ItemSetVestmentsOfProphecy = core.NewItemSet(core.ItemSet{
	Name: "Vestments of Prophecy",
	Bonuses: map[int32]core.ApplyEffect{
		// -0.1 sec to the casting time of your Flash Heal spell.
		3: func(agent core.Agent) {
			// Nothing to do
		},
		// Improves your chance to get a critical strike with Holy spells by 2%.
		5: func(agent core.Agent) {
			priest := agent.(PriestAgent).GetPriest()
			priest.PseudoStats.SchoolBonusCritChance[stats.SchoolIndexHoly] += 2 * core.CritRatingPerCritChance
		},
		// Increases your chance of a critical hit with Prayer of Healing by 25%.
		8: func(agent core.Agent) {
			// Nothing to do
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 3 Item Sets - BWL
///////////////////////////////////////////////////////////////////////////

var ItemSetVestmentsOfTranscendence = core.NewItemSet(core.ItemSet{
	Name: "Vestments of Transcendence",
	Bonuses: map[int32]core.ApplyEffect{
		// Allows 15% of your Mana regeneration to continue while casting.
		3: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.PseudoStats.SpiritRegenRateCasting += .15
		},
		// When struck in melee there is a 50% chance you will Fade for 4 sec.
		5: func(agent core.Agent) {
			// Nothing to do
		},
		// Your Greater Heals now have a heal over time component equivalent to a rank 5 Renew.
		8: func(agent core.Agent) {
			// Nothing to do
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Classic Phase 4 Item Sets - ZG and AB
///////////////////////////////////////////////////////////////////////////

var ItemSetConfessorsRaiment = core.NewItemSet(core.ItemSet{
	Name: "Confessor's Raiment",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases healing done by spells and effects by up to 22.
		2: func(agent core.Agent) {
			c := agent.GetCharacter()
			c.AddStat(stats.HealingPower, 22)
		},
		// Increase the range of your Smite and Holy Fire spells by 5 yds.
		3: func(agent core.Agent) {
			// Nothing to do
		},
		// Reduces the casting time of your Mind Control spell by 0.5 sec.
		5: func(agent core.Agent) {
			// Nothing to do
		},
	},
})
