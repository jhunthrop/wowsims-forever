package rogue_test

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
)

const (
	grimstitchArmorSetID  int32 = 2099
	grimstitchFinisherRow int32 = 1301708 // 5P: Envenom and Eviscerate cost -5 Energy
)

func grimstitchRogue(t *testing.T, pieces int) *dpsrogue.DpsRogue {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassRogue,
			Race:               proto.Race_RaceHuman,
			Level:              60,
			Buffs:              core.FullBuffs.Player,
			DistanceFromTarget: 5,
		},
		&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}},
	)
	sim := clientsetbonustest.PrePulledSim(t, player, grimstitchArmorSetID, pieces)
	built, ok := sim.Raid.Parties[0].Players[0].(*dpsrogue.DpsRogue)
	if !ok {
		t.Fatal("player 0 did not build as a *dpsrogue.DpsRogue")
	}
	return built
}

// 2P (hit) and 4P (attack power against Humanoids) are flat rows.
func TestGrimstitchAutomaticBonusesMatchTheRows(t *testing.T) {
	bare := grimstitchRogue(t, 0).GetCharacter()
	for _, pieces := range []int{2, 4} {
		clientsetbonustest.AssertAutomaticTotals(t, grimstitchArmorSetID, pieces, bare, grimstitchRogue(t, pieces).GetCharacter())
	}
}

// 5P: "Reduces the cost of your Envenom and Eviscerate abilities by 5
// Energy". The engine has Eviscerate only.
func TestGrimstitchFivePieceLowersEviscerateCost(t *testing.T) {
	four := grimstitchRogue(t, 4).GetRogue().Eviscerate.Cost.GetCurrentCost()
	five := grimstitchRogue(t, 5).GetRogue().Eviscerate.Cost.GetCurrentCost()
	want := core.MustClientSpellRow(grimstitchFinisherRow).Effects[0].Points
	if got := five - four; got != want {
		t.Errorf("five pieces change Eviscerate's cost by %v, the row says %v", got, want)
	}
}
