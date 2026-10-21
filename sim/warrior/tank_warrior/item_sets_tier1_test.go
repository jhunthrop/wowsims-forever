package tankwarrior

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
)

func battleplateWarrior(t *testing.T, pieces int) *warrior.Warrior {
	t.Helper()
	player := &proto.Player{
		Name:          "Tank",
		Class:         proto.Class_ClassWarrior,
		Race:          proto.Race_RaceOrc,
		Level:         60,
		TalentsString: warrior.ForeverProtectionTalents,
		Consumes:      &proto.Consumes{},
		Buffs:         &proto.IndividualBuffs{},
		Spec:          PlayerOptionsBasic,
	}
	sim := clientsetbonustest.PrePulledSim(t, player, warrior.ItemSetBattleplateOfGlory.ID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(warrior.WarriorAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warrior agent")
	}
	return agent.GetWarrior()
}

// 2P (defense) and 4P (expertise) are flat rows.
func TestBattleplateAutomaticBonusesMatchTheRows(t *testing.T) {
	setID := warrior.ItemSetBattleplateOfGlory.ID
	bare := battleplateWarrior(t, 0).GetCharacter()
	for _, pieces := range []int{2, 4} {
		clientsetbonustest.AssertAutomaticTotals(t, setID, pieces, bare, battleplateWarrior(t, pieces).GetCharacter())
	}
}

// 5P: "Reduces the cooldown on your Shield Wall ability by 30 sec".
func TestBattleplateFivePieceShortensShieldWall(t *testing.T) {
	shieldWallRow := clientsetbonus.SpellAt(warrior.ItemSetBattleplateOfGlory.ID, clientsetbonus.FivePieces)
	want := time.Duration(core.MustClientSpellRow(shieldWallRow).Effects[0].Points) * time.Millisecond

	four := battleplateWarrior(t, 4)
	five := battleplateWarrior(t, 5)
	cooldown := func(w *warrior.Warrior) time.Duration {
		return w.GetCharacter().GetSpell(core.ActionID{SpellID: warrior.ShieldWallSpellId[0]}).CD.Duration
	}
	if got := cooldown(five) - cooldown(four); got != want {
		t.Errorf("five pieces change Shield Wall's cooldown by %v, the row says %v", got, want)
	}
}
