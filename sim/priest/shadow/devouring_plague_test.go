package shadow

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/priest"
)

// TestDevouringPlagueTopRankRegisteredAtMaxLevel guards against
// registerDevouringPlagueSpell's loop stopping one short of
// DevouringPlagueRanks again: a level-60 priest must have rank 6
// (19280, the real client's level-60 learn) registered, not just rank 5
// (19279).
func TestDevouringPlagueTopRankRegisteredAtMaxLevel(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          proto.Race_RaceUndead,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         core.FullBuffs.Player,
		TalentsString: P1Talents,
	}, PlayerOptionsBasic)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}

	env, _, _ := core.NewEnvironment(raid, encounter, true)
	agent := env.Raid.Parties[0].Players[0]
	spriest, ok := agent.(*ShadowPriest)
	if !ok {
		t.Fatalf("player agent is %T, want *ShadowPriest", agent)
	}

	topRank := spriest.DevouringPlague[priest.DevouringPlagueRanks]
	if topRank == nil {
		t.Fatalf("DevouringPlague[%d] not registered at level 60", priest.DevouringPlagueRanks)
	}
	if topRank.SpellID != priest.DevouringPlagueSpellId[priest.DevouringPlagueRanks] {
		t.Errorf("top rank SpellID = %d, want %d", topRank.SpellID, priest.DevouringPlagueSpellId[priest.DevouringPlagueRanks])
	}
}

// TestDevouringPlagueCooldownMatchesClient guards the conformance fix
// (sim/core/testdata/conformance/priest.golden.md flagged cooldown_ms
// 60000->180000, client->engine): every rank's client
// category_cooldown_ms is 60000 (1 min), not 3 min.
func TestDevouringPlagueCooldownMatchesClient(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          proto.Race_RaceUndead,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         core.FullBuffs.Player,
		TalentsString: P1Talents,
	}, PlayerOptionsBasic)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}

	env, _, _ := core.NewEnvironment(raid, encounter, true)
	agent := env.Raid.Parties[0].Players[0]
	spriest, ok := agent.(*ShadowPriest)
	if !ok {
		t.Fatalf("player agent is %T, want *ShadowPriest", agent)
	}

	topRank := spriest.DevouringPlague[priest.DevouringPlagueRanks]
	if topRank == nil {
		t.Fatalf("DevouringPlague[%d] not registered at level 60", priest.DevouringPlagueRanks)
	}
	if topRank.CD.Duration != time.Minute {
		t.Errorf("Devouring Plague CD.Duration = %s, want %s (client's category_cooldown_ms)", topRank.CD.Duration, time.Minute)
	}
}

// TestDevouringPlagueRank6ResolvesDistinctlyFromRank5ViaGetSpell is the
// rotation-accuracy program's engine-side check for Devouring Plague
// (lane engine-3, item 2): the site's ladder golden shows a level-60
// shadow priest casting only rank 5 (19279), never rank 6 (19280), even
// though data/curated/apl/priest-shadow.json's own notes say
// sim/request's rewrite resolves the Inner Focus sequence's authored id
// to 19280 at level 60.
//
// This test proves the engine side of that chain is not the cause:
// unit.GetSpell(ActionID{SpellID: 19280}) - the exact lookup
// APLRotation.GetAPLSpell uses to resolve a castSpell action - finds a
// *Spell distinct from rank 5's (19279), with rank 6's higher mana cost,
// exactly as TestDevouringPlagueTopRankRegisteredAtMaxLevel above already
// proves rank 6 is registered at all. The site's zero_casts finding is
// therefore rotation-side, not engine-side: the curated file's Inner
// Focus sequence (and its synced embedded APL,
// sim/request/apl/priest-shadow.apl.json) authors Devouring Plague's id
// as 19279 (rank 5) - a leftover from this file's own stale "Fix round
// 1" - and sim/request's rewrite of ranked spell ids is skipped at
// MaxLevel (rotation() in request.go), so the authored rank 5 id is what
// reaches the engine unchanged. Recommended rotation fix: change that
// line's "spellId": 19279 to 19280 (rank 6, the client's real top rank)
// and drop the matching expected_idle entry, mirroring the convention
// spellranks.go's own doc comment states ("The APL always references
// the highest, since it is authored for MaxLevel").
func TestDevouringPlagueRank6ResolvesDistinctlyFromRank5ViaGetSpell(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          proto.Race_RaceUndead,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         core.FullBuffs.Player,
		TalentsString: P1Talents,
	}, PlayerOptionsBasic)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}

	env, _, _ := core.NewEnvironment(raid, encounter, true)
	agent := env.Raid.Parties[0].Players[0]
	spriest, ok := agent.(*ShadowPriest)
	if !ok {
		t.Fatalf("player agent is %T, want *ShadowPriest", agent)
	}
	unit := &spriest.Priest.Character.Unit

	rank5 := unit.GetSpell(core.ActionID{SpellID: 19279})
	if rank5 == nil {
		t.Fatal("Devouring Plague rank 5 (19279) not registered at level 60")
	}
	rank6 := unit.GetSpell(core.ActionID{SpellID: 19280})
	if rank6 == nil {
		t.Fatal("Devouring Plague rank 6 (19280) not registered at level 60 - " +
			"this WOULD be the engine bug the site diagnosed, but it is not " +
			"what's happening: see rank5's non-nil result above")
	}
	if rank5 == rank6 {
		t.Fatal("rank 5 and rank 6 Devouring Plague resolved to the same *Spell object")
	}

	rank5Cost := rank5.Cost.GetCurrentCost()
	rank6Cost := rank6.Cost.GetCurrentCost()
	if rank6Cost <= rank5Cost {
		t.Errorf("rank 6 Devouring Plague mana cost = %v, want more than rank 5's %v", rank6Cost, rank5Cost)
	}
}
