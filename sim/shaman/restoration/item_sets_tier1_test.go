package restoration

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/healsim"
	"github.com/wowsims/classic/sim/shaman"
)

var spiritcallerSetID = shaman.ItemSetTheSpiritcaller.ID

// spiritcallerShaman is a restoration shaman wearing pieces of The
// Spiritcaller.
func spiritcallerShaman(t *testing.T, race proto.Race, talents string, pieces int) *RestorationShaman {
	t.Helper()
	player := newHealerPlayer(60, talents, nil)
	player.Race = race
	clientsetbonustest.Wear(player, spiritcallerSetID, pieces)
	sim := core.NewSim(healsim.Request(player, quietRaid(), 60, 1), simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	healer, ok := sim.Raid.Parties[0].Players[healsim.HealerIndex].(*RestorationShaman)
	if !ok {
		t.Fatal("the raid's first player is not a Restoration shaman")
	}
	return healer
}

// The flat bonuses are read on an Orc with no talents, so no racial or
// talent multiplier scales the set's amounts.
func TestSpiritcallerFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, spiritcallerSetID, func(pieces int) *core.Character {
		return spiritcallerShaman(t, proto.Race_RaceOrc, "", pieces).GetCharacter()
	})
}

// 5P: "Reduces the cooldown on your Riptide spell by 1 sec".
func TestSpiritcallerFivePieceShortensRiptide(t *testing.T) {
	clientsetbonustest.AssertCooldownBonus(t, spiritcallerSetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		healer := spiritcallerShaman(t, proto.Race_RaceTroll, riptideOnly, pieces)
		return clientsetbonustest.SpellWithMask(t, &healer.Unit, shaman.ShamanSpellMaskRiptide)
	})
}
