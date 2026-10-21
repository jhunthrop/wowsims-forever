package restoration

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
)

var grovekeeperRaimentSetID = druid.ItemSetGrovekeeperRaiment.ID

// grovekeeperRaimentDruid is a restoration druid wearing pieces of
// Grovekeeper Raiment.
func grovekeeperRaimentDruid(t *testing.T, race proto.Race, talents string, pieces int) *RestorationDruid {
	t.Helper()
	player := newPlayer(60, talents, stats.Stats{}, nil)
	player.Race = race
	clientsetbonustest.Wear(player, grovekeeperRaimentSetID, pieces)
	resto, _ := newDruid(t, player)
	return resto
}

// The flat bonuses are read on a Night Elf with no talents, so no racial or
// talent multiplier scales the set's amounts.
func TestGrovekeeperRaimentFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, grovekeeperRaimentSetID, func(pieces int) *core.Character {
		return grovekeeperRaimentDruid(t, proto.Race_RaceNightElf, "", pieces).GetCharacter()
	})
}

// 5P: "Reduces the cooldown on your Swiftmend spell by 3 sec".
func TestGrovekeeperRaimentFivePieceShortensSwiftmend(t *testing.T) {
	talents := talentsString(t, map[string]int{"swiftmend": 1})
	clientsetbonustest.AssertCooldownBonus(t, grovekeeperRaimentSetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		return grovekeeperRaimentDruid(t, proto.Race_RaceTauren, talents, pieces).Swiftmend.Spell
	})
}
