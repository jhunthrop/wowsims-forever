package warlock

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// Demonheart Raiment is Forever's Tier 1 warlock set (client ItemSet 2100,
// build 1.60.1.70009). The 2-piece hit and the 4-piece spell damage against
// Demons are flat bonuses applied from the client's rows.
const (
	demonheartRaimentSetID int32 = 2100
	demonheartBanishBonus  int32 = 1301068
	demonheartLifeTapBonus int32 = 1301715
)

var ItemSetDemonheartRaiment = core.NewClientItemSet(core.ClientSetModel{
	ID: demonheartRaimentSetID,
	Effects: map[int32]core.ApplyEffect{
		// Your Life Tap generates 20% more Mana at no additional Health cost.
		demonheartLifeTapBonus: func(agent core.Agent) {
			agent.(WarlockAgent).GetWarlock().lifeTapManaBonus = clientsetbonus.DummyPercent(demonheartLifeTapBonus)
		},
	},
	NoSim: map[int32]string{
		demonheartBanishBonus: "shortens the cast time of Banish, crowd control the sim never casts",
	},
})
