package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

var justiceBattlegearSetID = paladin.ItemSetJusticeBattlegear.ID

// justiceBattlegearPaladin is a retribution paladin wearing pieces of
// Justice Battlegear.
func justiceBattlegearPaladin(t *testing.T, race proto.Race, pieces int) *paladin.Paladin {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               race,
		Level:              60,
		Buffs:              core.FullBuffs.Player,
		DistanceFromTarget: 5,
	}, PlayerOptionsSealofRighteousness)
	sim := clientsetbonustest.PrePulledSim(t, player, justiceBattlegearSetID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(paladin.PaladinAgent)
	if !ok {
		t.Fatal("the raid's first player is not a paladin")
	}
	return agent.GetPaladin()
}

// The flat bonuses are read on a Dwarf: the racial bonuses of other races
// scale the set's amounts.
func TestJusticeBattlegearFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, justiceBattlegearSetID, func(pieces int) *core.Character {
		return &justiceBattlegearPaladin(t, proto.Race_RaceDwarf, pieces).Character
	})
}

// 5P: "Reduces the cooldown on your Judgement spell by 0.5 sec".
func TestJusticeBattlegearFivePieceShortensJudgement(t *testing.T) {
	clientsetbonustest.AssertCooldownBonus(t, justiceBattlegearSetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		pal := justiceBattlegearPaladin(t, proto.Race_RaceHuman, pieces)
		return clientsetbonustest.SpellWithMask(t, &pal.Unit, paladin.PaladinSpellMaskJudgement)
	})
}
