package item_sets

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Keep these in alphabetical order.

// https://www.wowhead.com/classic/item-set=491/blue-dragon-mail
var ItemSetBlueDragonMail = core.NewItemSet(core.ItemSet{
	Name: "Blue Dragon Mail",
	ID:   491,
	Bonuses: map[int32]core.ApplyEffect{
		// +4 All Resistances.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddResistances(4)
		},
		// Increases damage and healing done by magical spells and effects by up to 28.
		3: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStat(stats.SpellPower, 28)
		},
	},
})

// https://www.wowhead.com/classic/item-set=443/bloodsoul-embrace
var ItemSetBloodsoulEmbrace = core.NewItemSet(core.ItemSet{
	Name: "Bloodsoul Embrace",
	Bonuses: map[int32]core.ApplyEffect{
		// Restores 12 mana per 5 sec.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStat(stats.MP5, 12)
		},
	},
})

// https://www.wowhead.com/classic/item-set=421/bloodvine-garb
var ItemSetBloodvineGarb = core.NewItemSet(core.ItemSet{
	Name: "Bloodvine Garb",
	Bonuses: map[int32]core.ApplyEffect{
		// Improves your chance to get a critical strike with spells by 2%.
		3: func(agent core.Agent) {
			character := agent.GetCharacter()
			if character.HasProfession(proto.Profession_Tailoring) {
				character.AddStat(stats.Crit, 2*core.CritRatingPerCritChance)
			}
		},
	},
})

// https://www.wowhead.com/classic/item-set=442/blood-tiger-harness
var ItemSetBloodTigerHarness = core.NewItemSet(core.ItemSet{
	Name: "Blood Tiger Harness",
	Bonuses: map[int32]core.ApplyEffect{
		// Improves your chance to get a critical strike by 1%.
		// Improves your chance to get a critical strike with spells by 1%.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			// Forever: merged from MeleeCrit 1% + SpellCrit 1%. One effect
			// under a unified stat gets one write.
			character.AddStat(stats.Crit, 1*core.CritRatingPerCritChance)
		},
	},
})

// https://www.wowhead.com/classic/item-set=143/devilsaur-armor
var ItemSetDevilsaurArmor = core.NewItemSet(core.ItemSet{
	Name: "Devilsaur Armor",
	ID:   143,
	Bonuses: map[int32]core.ApplyEffect{
		// Improves your chance to hit by 2%.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStat(stats.Hit, 2*core.HitRatingPerHitChance)
		},
	},
})

// https://www.wowhead.com/classic/item-set=490/green-dragon-mail
var ItemSetGreenDragonMail = core.NewItemSet(core.ItemSet{
	Name: "Green Dragon Mail",
	ID:   490,
	Bonuses: map[int32]core.ApplyEffect{
		// Restores 3 mana per 5 sec.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStat(stats.MP5, 3)
		},
		// Allows 15% of your Mana regeneration to continue while casting.
		3: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.PseudoStats.SpiritRegenRateCasting += .15
		},
	},
})

// https://www.wowhead.com/classic/item-set=144/ironfeather-armor
var ItemSetIronfeatherArmor = core.NewItemSet(core.ItemSet{
	Name: "Ironfeather Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases damage and healing done by magical spells and effects by up to 20.
		2: func(agent core.Agent) {
			character := agent.GetCharacter()
			character.AddStat(stats.SpellPower, 20)
		},
	},
})
