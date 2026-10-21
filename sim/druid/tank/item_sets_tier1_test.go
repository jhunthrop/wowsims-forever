package tank

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/druid"
)

var grovekeeperRageSetID = druid.ItemSetGrovekeeperRage.ID

// grovekeeperRageBear is a bear tank wearing pieces of Grovekeeper Rage.
func grovekeeperRageBear(t *testing.T, race proto.Race, talents string, pieces int) *FeralTankDruid {
	t.Helper()
	player := bearPlayer(60, talents, nil, nil)
	player.Race = race
	player.Buffs = &proto.IndividualBuffs{}
	clientsetbonustest.Wear(player, grovekeeperRageSetID, pieces)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	raid.Tanks = append(raid.Tanks, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  harnessEncounter(),
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	bear, ok := sim.Raid.Parties[0].Players[0].(*FeralTankDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *FeralTankDruid")
	}
	return bear
}

// The flat bonuses are read on a Night Elf with no talents, so no racial or
// talent multiplier scales the set's amounts.
func TestGrovekeeperRageFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, grovekeeperRageSetID, func(pieces int) *core.Character {
		return grovekeeperRageBear(t, proto.Race_RaceNightElf, "", pieces).GetCharacter()
	})
}

// 5P: "Reduces the cooldown on your Berserk ability by 15 sec".
func TestGrovekeeperRageFivePieceShortensBerserk(t *testing.T) {
	talents := talentString(t, map[string]int{"berserk": 1})
	clientsetbonustest.AssertCooldownBonus(t, grovekeeperRageSetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		return grovekeeperRageBear(t, proto.Race_RaceTauren, talents, pieces).Berserk.Spell
	})
}
