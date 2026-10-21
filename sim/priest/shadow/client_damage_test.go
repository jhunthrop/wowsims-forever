package shadow

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/priest"
)

// TestShadowPriestDeclaresTheClientRollAtItsLevel checks the wiring: the
// roll each registered top-rank damage spell declares is the client's at
// the priest's level (60).
func TestShadowPriestDeclaresTheClientRollAtItsLevel(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          proto.Race_RaceUndead,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         core.FullBuffs.Player,
		TalentsString: P1Talents,
	}, PlayerOptionsBasic)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{Duration: 10, Targets: []*proto.Target{core.NewDefaultTarget()}}, true)
	spriest := env.Raid.Parties[0].Players[0].(*ShadowPriest)
	client := clientdamagetest.LoadClient(t, "../..", "priest")

	declared := map[string]*core.Spell{
		"Shadow Word: Pain":  spriest.ShadowWordPain[priest.ShadowWordPainRanks],
		"Mind Blast":         spriest.MindBlast[priest.MindBlastRanks],
		"Shadow Word: Death": spriest.ShadowWordDeath[priest.ShadowWordDeathRanks],
		"Devouring Plague":   spriest.DevouringPlague[priest.DevouringPlagueRanks],
	}
	if len(spriest.MindFlay) > 0 { // talent-gated; the P1 build may not take it
		declared["Mind Flay"] = spriest.MindFlay[priest.MindFlayRanks][0]
	}
	for name, spell := range declared {
		if spell == nil {
			t.Fatalf("%s top rank is not registered at level 60", name)
		}
		clientdamagetest.AssertRoll(t, client, name, spell.ActionID.SpellID, 0, 60, spell.ClientBaseDamage)
	}
}
