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
