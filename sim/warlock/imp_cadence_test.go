package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Sim-accuracy 2026-11-12 (warlock logs lane): a live Imp lands a Firebolt
// every 2.43 s, not every 2.2 s. Three Imps on foreverlogs.gg after the
// 10 Nov hotfix (reports 2847 and 2850, 674 hit-to-hit gaps at a standing
// target: medians 2.423, 2.431 and 2.431 s, quartiles 2.41 to 2.46 s) all
// read the same, while a Succubus swings on an exact 2.0 s timer. The cast
// is 2 s, so the pet AI's delay between casts is about 430 ms, not the 200
// ms the engine carried from upstream.
func TestImpFireboltCadenceMatchesTheLogs(t *testing.T) {
	const fightSeconds = 300
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: "",
			Rotation:      &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor: proto.WarlockOptions_NoArmor, Summon: proto.WarlockOptions_Imp, WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
			PetPoolMana: true,
		}}},
	)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: fightSeconds, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{Iterations: 3, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %v", result.Error.Message)
	}
	var imp *proto.UnitMetrics
	for _, pet := range result.RaidMetrics.Parties[0].Players[0].Pets {
		if pet.Name == "Imp" {
			imp = pet
		}
	}
	if imp == nil {
		t.Fatal("no Imp in the result")
	}
	casts := float64(castsOf(imp, 11763)) / 3
	wantLow, wantHigh := fightSeconds/2.46, fightSeconds/2.41
	if casts < wantLow || casts > wantHigh {
		t.Errorf("the Imp cast Firebolt %.1f times in %d s (one every %.2f s), want one every 2.41 to 2.46 s (%.0f to %.0f casts)", casts, fightSeconds, fightSeconds/casts, wantLow, wantHigh)
	}
}
