package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever demons scale from their summoner's stats (Blizzard's warlock deep
// dive, 2026-10-07) through Warlock Pet Scaling 416189, the same
// server-computed channels as Hunter Pet Scaling 415429, and the hunter deep
// dive says such pets "can no longer receive player buffs that increase their
// stats". A level-60 Succubus summoned under every raid buff keeps its own
// base sheet: Strength 129, Agility 85 and 129 x 2 - 20 = 238 attack power.
func TestDemonReceivesNoPlayerBuffs(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassWarlock,
			Race:      proto.Race_RaceOrc,
			Level:     60,
			Equipment: &proto.EquipmentSpec{},
			Buffs:     core.FullBuffs.Player,
		},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor:  proto.WarlockOptions_NoArmor,
			Summon: proto.WarlockOptions_Succubus,
		}}},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	agent, ok := env.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("player 0 did not build as a warlock")
	}
	got := agent.GetWarlock().Succubus.GetStats()
	if got[stats.Strength] != 129 || got[stats.Agility] != 85 {
		t.Errorf("Succubus Strength/Agility = %.1f/%.1f, want the unbuffed 129/85", got[stats.Strength], got[stats.Agility])
	}
	if math.Abs(got[stats.AttackPower]-238) > 1e-6 {
		t.Errorf("Succubus attack power = %.1f, want the unbuffed 238", got[stats.AttackPower])
	}
}
