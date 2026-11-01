package mage

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestArcaneMissilesTopRankRegisteredAtMaxLevel: a level-60 mage has the
// top trainer rank, rank 7 (10212). Rank 8 (25345) is taught by Tome of
// Arcane Missiles VIII, an Ahn'Qiraj book (warcraft.wiki.gg "Arcane
// Missiles"; wowclassicdb's Ruins of Ahn'Qiraj loot), so like Frostbolt 11
// and Fireball 12 it registers only with core.IncludeAQ.
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

	want := core.TernaryInt(core.IncludeAQ, ArcaneMissilesRanks, ArcaneMissilesRanks-1)
	topRank := mage.ArcaneMissiles[want]
	if topRank == nil {
		t.Fatalf("ArcaneMissiles[%d] not registered at level 60", want)
	}
	if topRank.SpellID != ArcaneMissilesChannelSpellId[want] {
		t.Errorf("top rank SpellID = %d, want %d", topRank.SpellID, ArcaneMissilesChannelSpellId[want])
	}
	if !core.IncludeAQ && mage.ArcaneMissiles[ArcaneMissilesRanks] != nil {
		t.Errorf("the Ahn'Qiraj book rank %d (25345) is registered without core.IncludeAQ", ArcaneMissilesRanks)
	}
}
