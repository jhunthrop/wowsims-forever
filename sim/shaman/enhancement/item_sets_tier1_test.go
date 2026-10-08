package enhancement

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/shaman"
)

var spiritcallersRageSetID = shaman.ItemSetTheSpiritcallersRage.ID

// stormstrikeNode is Stormstrike's position in the Enhancement tree.
const stormstrikeNode = 12

// spiritcallersRageShaman is an enhancement shaman wearing pieces of The
// Spiritcaller's Rage.
func spiritcallersRageShaman(t *testing.T, race proto.Race, talents string, pieces int) *shaman.Shaman {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassShaman,
		Race:          race,
		Level:         60,
		Buffs:         core.FullBuffs.Player,
		TalentsString: talents,
	}, &proto.Player_EnhancementShaman{
		EnhancementShaman: &proto.EnhancementShaman{
			Options: &proto.EnhancementShaman_Options{SyncType: proto.ShamanSyncType_Auto},
		},
	})
	sim := clientsetbonustest.PrePulledSim(t, player, spiritcallersRageSetID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	return agent.GetShaman()
}

// The flat bonuses are read on an Orc with no talents, so no racial or
// talent multiplier scales the set's amounts.
func TestSpiritcallersRageFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, spiritcallersRageSetID, func(pieces int) *core.Character {
		return spiritcallersRageShaman(t, proto.Race_RaceOrc, "", pieces).GetCharacter()
	})
}

// 5P: "Reduces the cooldown on your Stormstrike ability by 0.5 sec".
func TestSpiritcallersRageFivePieceShortensStormstrike(t *testing.T) {
	talents := shamanTalentString(nil, map[int]int{stormstrikeNode: 1}, nil)
	clientsetbonustest.AssertCooldownBonus(t, spiritcallersRageSetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		return spiritcallersRageShaman(t, proto.Race_RaceOrc, talents, pieces).Stormstrike
	})
}
