package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// runImpOnly runs a bare level-60 warlock who casts nothing while the Imp
// fires at a boss for fightSeconds.
func runImpOnly(t *testing.T, fightSeconds int, iterations int) *proto.UnitMetrics {
	t.Helper()
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
		}}},
	)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: float64(fightSeconds), Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{Iterations: int32(iterations), RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %v", result.Error.Message)
	}
	for _, pet := range result.RaidMetrics.Parties[0].Players[0].Pets {
		if pet.Name == "Imp" {
			return pet
		}
	}
	t.Fatal("no Imp in the result")
	return nil
}

func fireboltMetrics(imp *proto.UnitMetrics) *proto.ActionMetrics {
	for _, action := range imp.Actions {
		if action.Id.GetSpellId() == 11763 {
			return action
		}
	}
	return nil
}

// Sim-accuracy 2026-11-12 (warlock logs lane): a live Imp lands a Firebolt
// every 2.43 s, not every 2.2 s. Three Imps on foreverlogs.gg after the
// 10 Nov hotfix (reports 2847 and 2850, 674 hit-to-hit gaps at a standing
// target: medians 2.423, 2.431 and 2.431 s, quartiles 2.41 to 2.46 s) all
// read the same, while a Succubus swings on an exact 2.0 s timer. The cast
// is 2 s, so the pet AI's delay between casts is about 430 ms, not the 200
// ms the engine carried from upstream. The fight is short enough that the
// bare Imp's mana lasts it (its pool is about nine Firebolts at 60; 80 s
// and 300 s both left it waiting on regen).
func TestImpFireboltCadenceMatchesTheLogs(t *testing.T) {
	const fightSeconds = 20
	imp := runImpOnly(t, fightSeconds, 3)
	casts := float64(castsOf(imp, 11763)) / 3
	// The first cast is at the pull, then one every 2.43 s: 9.2, so 9. At
	// 2.2 s it is 10.1, so 10.
	want := 1 + fightSeconds/2.43
	if casts < want-0.6 || casts > want+0.6 {
		t.Errorf("the Imp cast Firebolt %.1f times in %d s, want about %.1f (one every 2.43 s)", casts, fightSeconds, want)
	}
}

// Thirteen live Imps (foreverlogs.gg 2816 to 2853, 5 to 12 Nov, owners at
// 12 to 40, 3,501 Firebolts) crit 5.9% of their Firebolts, each between 3.6
// and 7.7%, where the engine's 2.2% came from the beta logs. The warlock
// wears no gear here, so the rate is the Imp's own.
func TestImpFireboltCritIsTheLiveRate(t *testing.T) {
	firebolt := fireboltMetrics(runImpOnly(t, 80, 60))
	if firebolt == nil {
		t.Fatal("the Imp cast no Firebolt")
	}
	var hits, crits int32
	for _, target := range firebolt.Targets {
		hits += target.Hits + target.Crits
		crits += target.Crits
	}
	rate := 100 * float64(crits) / float64(hits)
	if rate < 4.5 || rate > 7.5 {
		t.Errorf("Firebolt crit %.1f%% of %d hits, want about 5.9%% (13 live Imps)", rate, hits)
	}
}
