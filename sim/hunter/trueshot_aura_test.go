package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

func buildHunterWithRaidBuffs(t *testing.T, raidBuffs *proto.RaidBuffs) *Hunter {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassHunter,
			Race:      proto.Race_RaceOrc,
			Level:     60,
			Equipment: &proto.EquipmentSpec{},
			Buffs:     &proto.IndividualBuffs{},
		},
		&proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{PetType: proto.Hunter_Options_PetNone}}},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, raidBuffs, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
}

// Every Trueshot Aura rank in the client is one effect, aura 124 (ranged
// attack power): 1299346, 1299348, 19506, 20905, 20906. Melee attack
// power would be aura 99. Melee specs gain nothing from it.
func TestTrueshotAuraIsRangedAttackPowerOnly(t *testing.T) {
	with := buildHunterWithRaidBuffs(t, &proto.RaidBuffs{TrueshotAura: true})
	without := buildHunterWithRaidBuffs(t, &proto.RaidBuffs{})
	if got := with.GetStat(stats.AttackPower) - without.GetStat(stats.AttackPower); got != 0 {
		t.Errorf("Trueshot Aura adds %.1f melee attack power, want 0", got)
	}
	if got := with.GetStat(stats.RangedAttackPower) - without.GetStat(stats.RangedAttackPower); got <= 0 {
		t.Errorf("Trueshot Aura adds %.1f ranged attack power, want the rank's amount", got)
	}
}
