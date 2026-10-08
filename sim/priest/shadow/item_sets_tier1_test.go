package shadow

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/priest"
)

var raimentsSetID = priest.ItemSetRaimentsOfConviction.ID

// raimentsPriest is a shadow priest wearing pieces of Raiments of
// Conviction.
func raimentsPriest(t *testing.T, race proto.Race, talents string, pieces int) *ShadowPriest {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          race,
		Level:         60,
		Buffs:         core.FullBuffs.Player,
		TalentsString: talents,
	}, PlayerOptionsBasic)
	sim := clientsetbonustest.PrePulledSim(t, player, raimentsSetID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(*ShadowPriest)
	if !ok {
		t.Fatal("the raid's first player is not a shadow priest")
	}
	return agent
}

// The flat bonuses are read on a Dwarf with no talents, so no racial or
// talent multiplier scales the set's amounts.
func TestRaimentsOfConvictionFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, raimentsSetID, func(pieces int) *core.Character {
		return raimentsPriest(t, proto.Race_RaceDwarf, "", pieces).GetCharacter()
	})
}

// 5P: "Reduces the cooldown on your Devouring Plague spell by 60 sec".
func TestRaimentsOfConvictionFivePieceShortensDevouringPlague(t *testing.T) {
	clientsetbonustest.AssertCooldownBonus(t, raimentsSetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		return raimentsPriest(t, proto.Race_RaceUndead, P1Talents, pieces).DevouringPlague[priest.DevouringPlagueRanks]
	})
}
