package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Push 113887 (13 November 2026) rebuilt Windfury Totem's attack. The rank 3
// buff spell 10610 lost its 246 attack power aura (99) and its extra attack
// (effect 19) and became a main-hand weapon strike on the target (effect
// 58, SpellMisc Attributes_3 0x400 main hand and 0x200 triggered can proc,
// SpellCategories DefenseType 2, category 2619 at 100 ms); the totem's
// aura 10612 gained a dummy (aura 4) carrying the 246. Our 13 November
// report logs it the same way: the totem's rank 2 hits as SPELL_DAMAGE and
// SPELL_MISSED (dodge) at the moment of the swing or Stormstrike that
// triggered it, where every report before the push logs
// SPELL_EXTRA_ATTACKS. So a proc is one yellow main-hand strike: no
// glancing blow, no extra white swing, no rage from a white swing.
func TestWindfuryTotemProcsAMainHandStrike(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassWarrior,
		Race:          proto.Race_RaceOrc,
		Level:         60,
		TalentsString: P1Talents,
		Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{
			{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			{Id: 12784}, {}, {},
		}},
		Consumes: &proto.Consumes{},
		Buffs:    &proto.IndividualBuffs{},
		Rotation: &proto.APLRotation{},
	}, PlayerOptionsFury)

	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{WindfuryTotem: true}, &proto.Debuffs{})
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 120, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 20, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	metrics := result.RaidMetrics.Parties[0].Players[0]

	for _, row := range metrics.ExtraAttacks {
		if row.Id.GetSpellId() == 10610 {
			t.Errorf("Windfury Totem granted %d extra white swings; since push 113887 it strikes instead", row.Attacks)
		}
	}

	var strikes, glances int32
	for _, action := range metrics.Actions {
		if action.Id.GetSpellId() != 10610 {
			continue
		}
		for _, target := range action.Targets {
			strikes += target.Hits
			glances += target.Glances
		}
	}
	if strikes == 0 {
		t.Fatal("Windfury Totem never struck: no landed 10610 hit in 20 fights")
	}
	if glances != 0 {
		t.Errorf("Windfury Totem's strike glanced %d times; a yellow strike never glances", glances)
	}
}
