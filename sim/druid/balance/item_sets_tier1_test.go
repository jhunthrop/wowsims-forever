package balance

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

var grovekeeperEclipseSetID = druid.ItemSetGrovekeeperEclipse.ID

// grovekeeperEclipseDruid is a moonkin wearing pieces of Grovekeeper
// Eclipse, and the boss it fights.
func grovekeeperEclipseDruid(t *testing.T, race proto.Race, talents string, pieces int) (*BalanceDruid, *core.Unit) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassDruid,
		Race:               race,
		Level:              60,
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talents,
		DistanceFromTarget: 30,
	}, PlayerOptionsAdaptive)
	sim := clientsetbonustest.PrePulledSim(t, player, grovekeeperEclipseSetID, pieces)
	moonkin, ok := sim.Raid.Parties[0].Players[0].(*BalanceDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *BalanceDruid")
	}
	return moonkin, sim.Encounter.TargetUnits[0]
}

// The flat bonuses are read on a Night Elf with no talents, so no racial or
// talent multiplier scales the set's amounts.
func TestGrovekeeperEclipseFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, grovekeeperEclipseSetID, func(pieces int) *core.Character {
		moonkin, _ := grovekeeperEclipseDruid(t, proto.Race_RaceNightElf, "", pieces)
		return moonkin.GetCharacter()
	})
}

func insectSwarmTicks(t *testing.T, pieces int) int32 {
	t.Helper()
	talents, err := core.TalentsStringFromRanks((&proto.DruidTalents{}).ProtoReflect(), druid.TalentTreeSizes, map[string]int{"insect_swarm": 1})
	if err != nil {
		t.Fatal(err)
	}
	moonkin, boss := grovekeeperEclipseDruid(t, proto.Race_RaceTauren, talents, pieces)
	rank := moonkin.InsectSwarm[druid.InsectSwarmRanks]
	return rank.Dot(boss).NumberOfTicks
}

// 5P: "Increases the duration of your Insect Swarm spell by 3 sec". Its
// damage lands on 2 second ticks, so the 3 seconds add one whole tick.
func TestGrovekeeperEclipseFivePieceLengthensInsectSwarm(t *testing.T) {
	extra := time.Duration(core.MustClientSpellRow(clientsetbonus.SpellAt(grovekeeperEclipseSetID, clientsetbonus.FivePieces)).Effects[0].Points) * time.Millisecond / druid.InsectSwarmTickLength
	bare := insectSwarmTicks(t, 0)
	if short := insectSwarmTicks(t, int(clientsetbonus.FivePieces)-1); short != bare {
		t.Errorf("four pieces change Insect Swarm from %d ticks to %d", bare, short)
	}
	if got, want := insectSwarmTicks(t, int(clientsetbonus.FivePieces)), bare+int32(extra); got != want {
		t.Errorf("five pieces give %d Insect Swarm ticks, want %d", got, want)
	}
}
