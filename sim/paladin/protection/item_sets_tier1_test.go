package protection

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

var justiceBattleplateSetID = paladin.ItemSetJusticeBattleplate.ID

// justiceBattleplatePaladin is a protection paladin wearing pieces of
// Justice Battleplate.
func justiceBattleplatePaladin(t *testing.T, race proto.Race, pieces int) *paladin.Paladin {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassPaladin,
		Race:  race,
		Level: 60,
		Buffs: core.FullBuffs.Player,
	}, PlayerOptionsRighteousFury)
	sim := clientsetbonustest.PrePulledSim(t, player, justiceBattleplateSetID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(paladin.PaladinAgent)
	if !ok {
		t.Fatal("the raid's first player is not a paladin")
	}
	return agent.GetPaladin()
}

// The flat bonuses are read on a Dwarf: the racial bonuses of other races
// scale the set's amounts.
func TestJusticeBattleplateFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, justiceBattleplateSetID, func(pieces int) *core.Character {
		return &justiceBattleplatePaladin(t, proto.Race_RaceDwarf, pieces).Character
	})
}

func forbearanceDuration(t *testing.T, pieces int) time.Duration {
	t.Helper()
	aura := justiceBattleplatePaladin(t, proto.Race_RaceHuman, pieces).GetAura("Forbearance")
	if aura == nil {
		t.Fatal("the paladin has no Forbearance aura")
	}
	return aura.Duration
}

// 5P: "Reduces the duration of Forbearance any time you gain it by 10 sec".
func TestJusticeBattleplateFivePieceShortensForbearance(t *testing.T) {
	change := clientsetbonus.DummyDuration(clientsetbonus.SpellAt(justiceBattleplateSetID, clientsetbonus.FivePieces))
	bare := forbearanceDuration(t, 0)
	if short := forbearanceDuration(t, int(clientsetbonus.FivePieces)-1); short != bare {
		t.Errorf("four pieces change Forbearance from %v to %v", bare, short)
	}
	if got, want := forbearanceDuration(t, int(clientsetbonus.FivePieces)), bare+change; got != want {
		t.Errorf("five pieces give a Forbearance of %v, want %v (%v %+v)", got, want, bare, change)
	}
}
