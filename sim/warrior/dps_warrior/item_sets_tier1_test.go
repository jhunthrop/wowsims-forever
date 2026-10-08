package dpswarrior

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
)

func battlegearWarrior(t *testing.T, pieces int) *warrior.Warrior {
	t.Helper()
	player := &proto.Player{
		Name:          "Warrior",
		Class:         proto.Class_ClassWarrior,
		Race:          proto.Race_RaceOrc,
		Level:         60,
		TalentsString: warrior.ForeverFuryTalents,
		Consumes:      &proto.Consumes{},
		Buffs:         &proto.IndividualBuffs{},
		Spec:          PlayerOptionsFury,
	}
	sim := clientsetbonustest.PrePulledSim(t, player, warrior.ItemSetBattlegearOfGlory.ID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(warrior.WarriorAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warrior agent")
	}
	return agent.GetWarrior()
}

// 2P (haste) and 4P (attack power against Humanoids) are flat rows.
func TestBattlegearAutomaticBonusesMatchTheRows(t *testing.T) {
	setID := warrior.ItemSetBattlegearOfGlory.ID
	bare := battlegearWarrior(t, 0).GetCharacter()
	for _, pieces := range []int{2, 4} {
		clientsetbonustest.AssertAutomaticTotals(t, setID, pieces, bare, battlegearWarrior(t, pieces).GetCharacter())
	}
}

// 5P: "Reduces the cooldown on your Recklessness ability by 30 sec".
func TestBattlegearFivePieceShortensRecklessness(t *testing.T) {
	reckRow := clientsetbonus.SpellAt(warrior.ItemSetBattlegearOfGlory.ID, clientsetbonus.FivePieces)
	want := time.Duration(core.MustClientSpellRow(reckRow).Effects[0].Points) * time.Millisecond

	four := battlegearWarrior(t, 4)
	five := battlegearWarrior(t, 5)
	cooldown := func(w *warrior.Warrior) time.Duration {
		return w.GetCharacter().GetSpell(core.ActionID{SpellID: warrior.RecklessnessSpellId[0]}).CD.Duration
	}
	if got := cooldown(five) - cooldown(four); got != want {
		t.Errorf("five pieces change Recklessness's cooldown by %v, the row says %v", got, want)
	}
}
