package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TestPetActiveSecondsCountsOnlyTheSummonedDemon checks the result's pet
// uptime: a warlock registers all four demons as pets, but only the one
// it summons is out for the fight. The other three must read zero, not
// the fight's length.
func TestPetActiveSecondsCountsOnlyTheSummonedDemon(t *testing.T) {
	const fightSeconds = 60
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassWarlock,
			Race:      proto.Race_RaceOrc,
			Level:     60,
			Equipment: &proto.EquipmentSpec{},
			Buffs:     core.FullBuffs.Player,
		},
		&proto.Player_Warlock{
			Warlock: &proto.Warlock{
				Options: &proto.WarlockOptions{
					Armor:       proto.WarlockOptions_NoArmor,
					Summon:      proto.WarlockOptions_Succubus,
					WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
				},
			},
		},
	)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs),
		Encounter: &proto.Encounter{
			Duration: fightSeconds,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{Iterations: 20, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	metrics := result.RaidMetrics.Parties[0].Players[0]
	if metrics.PetActiveSecondsAvg != 0 {
		t.Errorf("player pet_active_seconds_avg = %v, want 0", metrics.PetActiveSecondsAvg)
	}
	if len(metrics.Pets) < 2 {
		t.Fatalf("want every demon registered as a pet, got %d", len(metrics.Pets))
	}
	for _, pet := range metrics.Pets {
		want := 0.0
		if pet.Name == "Succubus" {
			want = fightSeconds
		}
		if math.Abs(pet.PetActiveSecondsAvg-want) > 0.5 {
			t.Errorf("%s pet_active_seconds_avg = %v, want %v", pet.Name, pet.PetActiveSecondsAvg, want)
		}
	}
}
