package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestSealOfCommandRanksResolveDistinctlyAtMaxLevel is the rotation-accuracy
// program's engine-side check for Seal of Command (lane engine-3, item 1):
// the site's ladder golden shows a level-60 retribution paladin casting
// only rank 1's Seal of Command (20375) and rank 1's Judgement of Command
// (20467), never rank 5's (20920 / 20966), even though the curated
// rotation's rewrite resolves the authored id to 20920 at level 60.
//
// This test proves the engine side of that chain is NOT the cause: a
// level-60 paladin has all five ranks registered under distinct
// ActionIDs (soc.go's registerSealOfCommand loop only breaks when
// paladin.Level < rank.level, which is never true here), and
// unit.GetSpell - the exact lookup APLRotation.GetAPLSpell uses to
// resolve a castSpell action's ActionID - finds rank 5's Seal of
// Command spell distinctly from rank 1's, with rank 5's own (higher)
// mana cost. The site's own zero_casts finding is therefore rotation-side:
// the curated file (data/curated/apl/paladin-retribution.json) and its
// synced embedded APL (sim/request/apl/paladin-retribution.apl.json)
// author Seal of Command's id as 20375 (rank 1), and
// sim/request's rewrite of ranked spell ids is skipped at MaxLevel
// (rotation() in request.go: "at MaxLevel the rotation was authored for
// exactly the ranks it already carries, so skipping the rewrite is both
// an optimization and the guarantee..."), an assumption every OTHER
// rotation in this codebase satisfies by authoring the top rank's id.
// Recommended rotation fix: change every "spellId": 20375 for Seal of
// Command in that curated file (the castSpell action and both
// auraIsActive conditions in the seal-twisting lines) to 20920 (rank 5),
// mirroring the convention spellranks.go's own doc comment states ("The
// APL always references the highest, since it is authored for
// MaxLevel").
func TestSealOfCommandRanksResolveDistinctlyAtMaxLevel(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassPaladin,
			Race:      proto.Race_RaceHuman,
			Level:     60,
			Equipment: &proto.EquipmentSpec{},
			Buffs:     core.FullBuffs.Player,
		},
		&proto.Player_RetributionPaladin{
			RetributionPaladin: &proto.RetributionPaladin{
				Options: optionsSealOfCommand,
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}

	env, _, _ := core.NewEnvironment(raid, encounter, true)
	agent := env.Raid.Parties[0].Players[0]
	ret, ok := agent.(*RetributionPaladin)
	if !ok {
		t.Fatalf("player agent is %T, want *RetributionPaladin", agent)
	}
	unit := &ret.Paladin.Character.Unit

	rank1 := unit.GetSpell(core.ActionID{SpellID: 20375})
	if rank1 == nil {
		t.Fatal("Seal of Command rank 1 (20375) not registered at level 60")
	}
	rank5 := unit.GetSpell(core.ActionID{SpellID: 20920})
	if rank5 == nil {
		t.Fatal("Seal of Command rank 5 (20920) not registered at level 60 - " +
			"this WOULD be the engine bug the site diagnosed, but it is not " +
			"what's happening: see rank1's non-nil result above")
	}
	if rank1 == rank5 {
		t.Fatal("rank 1 and rank 5 Seal of Command resolved to the same *Spell object")
	}

	rank1Cost := rank1.Cost.GetCurrentCost()
	rank5Cost := rank5.Cost.GetCurrentCost()
	if rank5Cost <= rank1Cost {
		t.Errorf("rank 5 Seal of Command mana cost = %v, want more than rank 1's %v", rank5Cost, rank1Cost)
	}

	// The Judgement of Command spells each rank's aura triggers are
	// likewise distinct and separately registered.
	judgeRank1 := unit.GetSpell(core.ActionID{SpellID: 20467})
	judgeRank5 := unit.GetSpell(core.ActionID{SpellID: 20966})
	if judgeRank1 == nil || judgeRank5 == nil {
		t.Fatalf("Judgement of Command ranks not both registered: rank1=%v rank5=%v", judgeRank1, judgeRank5)
	}
	if judgeRank1 == judgeRank5 {
		t.Fatal("rank 1 and rank 5 Judgement of Command resolved to the same *Spell object")
	}
}
