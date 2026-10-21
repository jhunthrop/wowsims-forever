package feral

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

var grovekeeperFerocitySetID = druid.ItemSetGrovekeeperFerocity.ID

// grovekeeperFerocityDruid is a cat druid wearing pieces of Grovekeeper
// Ferocity.
func grovekeeperFerocityDruid(t *testing.T, race proto.Race, pieces int) *FeralDruid {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassDruid,
		Race:               race,
		Level:              60,
		Buffs:              core.FullBuffs.Player,
		DistanceFromTarget: 5,
		TalentsString:      feralTalentsString(t, map[string]int{"shifting_power": 1}),
	}, PlayerOptionsMonoCat)
	sim := clientsetbonustest.PrePulledSim(t, player, grovekeeperFerocitySetID, pieces)
	cat, ok := sim.Raid.Parties[0].Players[0].(*FeralDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *FeralDruid")
	}
	return cat
}

// The flat bonuses are read on a Night Elf with no talents, so no racial or
// talent multiplier scales the set's amounts.
func TestGrovekeeperFerocityFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, grovekeeperFerocitySetID, func(pieces int) *core.Character {
		return grovekeeperFerocityDruid(t, proto.Race_RaceNightElf, pieces).GetCharacter()
	})
}

// 5P: "Reduces the cooldown on your Shifting Power ability by 1 sec".
func TestGrovekeeperFerocityFivePieceShortensShiftingPower(t *testing.T) {
	clientsetbonustest.AssertCooldownBonus(t, grovekeeperFerocitySetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		return grovekeeperFerocityDruid(t, proto.Race_RaceTauren, pieces).ShiftingPower.Spell
	})
}
