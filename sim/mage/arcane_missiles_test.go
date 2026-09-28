package mage

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestArcaneMissilesTopRankRegisteredAtMaxLevel guards against the
// registerArcaneMissilesSpell loop stopping one short of
// ArcaneMissilesRanks again: a level-60 mage must have rank 8 (25345,
// the real client's level-56 learn) registered, not just rank 7
// (10212).
func TestArcaneMissilesTopRankRegisteredAtMaxLevel(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassMage,
		Race:          proto.Race_RaceTroll,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         core.FullBuffs.Player,
		TalentsString: ForeverFrostTalents,
	}, PlayerOptions)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}

	env, _, _ := core.NewEnvironment(raid, encounter, true)
	agent := env.Raid.Parties[0].Players[0]
	mage, ok := agent.(*Mage)
	if !ok {
		t.Fatalf("player agent is %T, want *Mage", agent)
	}

	topRank := mage.ArcaneMissiles[ArcaneMissilesRanks]
	if topRank == nil {
		t.Fatalf("ArcaneMissiles[%d] not registered at level 60", ArcaneMissilesRanks)
	}
	if topRank.SpellID != ArcaneMissilesSpellId[ArcaneMissilesRanks] {
		t.Errorf("top rank SpellID = %d, want %d", topRank.SpellID, ArcaneMissilesSpellId[ArcaneMissilesRanks])
	}
}
