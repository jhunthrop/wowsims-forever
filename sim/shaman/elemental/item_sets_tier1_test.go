package elemental

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/shaman"
)

var spiritcallersStormSetID = shaman.ItemSetTheSpiritcallersStorm.ID

// spiritcallersStormShaman is an elemental shaman wearing pieces of The
// Spiritcaller's Storm.
func spiritcallersStormShaman(t *testing.T, race proto.Race, talents string, pieces int) *shaman.Shaman {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassShaman,
		Race:          race,
		Level:         60,
		Buffs:         core.FullBuffs.Player,
		TalentsString: talents,
	}, &proto.Player_ElementalShaman{
		ElementalShaman: &proto.ElementalShaman{Options: &proto.ElementalShaman_Options{}},
	})
	sim := clientsetbonustest.PrePulledSim(t, player, spiritcallersStormSetID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	return agent.GetShaman()
}

// The flat bonuses are read on an Orc with no talents, so no racial or
// talent multiplier scales the set's amounts.
func TestSpiritcallersStormFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, spiritcallersStormSetID, func(pieces int) *core.Character {
		return spiritcallersStormShaman(t, proto.Race_RaceOrc, "", pieces).GetCharacter()
	})
}

// 5P: "Reduces the cooldown on your Lava Burst spell by 1 sec". No shaman
// can learn Lava Burst since client 1.60.1.70755, so the bonus is declared
// NoSim rather than modelled.
func TestSpiritcallersStormFivePieceIsNoSim(t *testing.T) {
	fivePiece := clientsetbonus.SpellAt(spiritcallersStormSetID, clientsetbonus.FivePieces)
	for _, model := range core.ClientSetModels()[spiritcallersStormSetID] {
		if model.SpellID == fivePiece {
			if model.Kind != core.ClientBonusNoSim {
				t.Errorf("5-piece %d is %v, want NoSim", fivePiece, model.Kind)
			}
			return
		}
	}
	t.Fatalf("5-piece %d is not among the set's models", fivePiece)
}
