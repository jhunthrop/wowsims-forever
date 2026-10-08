package healing

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/priest"
)

var vestmentsSetID = priest.ItemSetVestmentsOfConviction.ID

// vestmentsPriest is a healing priest wearing pieces of Vestments of
// Conviction.
func vestmentsPriest(t *testing.T, race proto.Race, talents string, pieces int) *HealingPriest {
	t.Helper()
	_, agent := wornPriest(t, vestmentsSetID, race, talents, pieces)
	return agent
}

// wornPriest is a healing priest wearing pieces of a client set.
func wornPriest(t *testing.T, setID int32, race proto.Race, talents string, pieces int) (*core.Simulation, *HealingPriest) {
	t.Helper()
	player := healer(60, talents, &proto.HealingPriest_Options{}, &proto.APLRotation{})
	player.Race = race
	clientsetbonustest.Wear(player, setID, pieces)
	return agentSim(t, player)
}

func topRank(spells []*core.Spell) *core.Spell {
	for rank := len(spells) - 1; rank > 0; rank-- {
		if spells[rank] != nil {
			return spells[rank]
		}
	}
	return nil
}

// The flat bonuses are read on a Dwarf with no talents: the Human spirit
// bonus and the spirit talents would scale the set's Spirit.
func TestVestmentsOfConvictionFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, vestmentsSetID, func(pieces int) *core.Character {
		return vestmentsPriest(t, proto.Race_RaceDwarf, "", pieces).GetCharacter()
	})
}

// 5P: "Reduces the cooldown on your Penance and Prayer of Mending spells".
func TestVestmentsOfConvictionFivePieceShortensPenance(t *testing.T) {
	clientsetbonustest.AssertCooldownBonus(t, vestmentsSetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		return topRank(vestmentsPriest(t, proto.Race_RaceHuman, DiscTalents, pieces).Penance)
	})
}

func TestVestmentsOfConvictionFivePieceShortensPrayerOfMending(t *testing.T) {
	clientsetbonustest.AssertCooldownBonus(t, vestmentsSetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		return topRank(vestmentsPriest(t, proto.Race_RaceHuman, HolyTalents, pieces).PrayerOfMending)
	})
}
