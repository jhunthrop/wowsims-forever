package rogue_test

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
)

// rogueSim builds a one-rogue sim at a fixed seed, so two builds roll the same
// weapon damage.
func rogueSim(t *testing.T, talentsStr, gearSet string) (*core.Simulation, *dpsrogue.DpsRogue) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassRogue,
			Race:               proto.Race_RaceHuman,
			Equipment:          core.GetGearSet("../../ui/rogue/gear_sets", gearSet).GearSet,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
		},
		&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}},
	)
	raid := core.SinglePlayerRaidProto(player, nil, nil, nil)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{{Level: 63}}},
		SimOptions: &proto.SimOptions{RandomSeed: 42, Iterations: 1},
	}, simsignals.CreateSignals())
	built, ok := sim.Raid.Parties[0].Players[0].(*dpsrogue.DpsRogue)
	if !ok {
		t.Fatal("player 0 did not build as a *dpsrogue.DpsRogue")
	}
	return sim, built
}

// Hemorrhage 16511 is client effect 121 (normalized weapon damage, SpellEffect
// 693307) under effect 31 at 100%, like Backstab and Sinister Strike: its
// attack power counts at the normalized speed (1.7 s for a dagger), not the
// weapon's own.
func TestHemorrhageUsesNormalizedWeaponDamage(t *testing.T) {
	const ap = 1000.0
	sim, built := rogueSim(t, zeroRogueTalents, "combat_sinister_strike_prebis")
	got := built.GetRogue().HemorrhageWeaponDamageForTest(sim, ap)
	refSim, ref := rogueSim(t, zeroRogueTalents, "combat_sinister_strike_prebis")
	mh := ref.GetRogue().AutoAttacks.MH()
	if mh.SwingSpeed == mh.NormalizedSwingSpeed {
		t.Fatalf("the test weapon swings at its normalized %.1f s, so it cannot tell the two apart", mh.SwingSpeed)
	}
	want := mh.CalculateNormalizedWeaponDamage(refSim, ap)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Hemorrhage weapon damage = %.3f, want the normalized %.3f", got, want)
	}
}

// Dual Wield Specialization 13715 is aura 122, the off-hand damage percent
// (+25% a rank in 70755): it raises the off-hand weapon's damage, so
// Mutilate's off-hand strike (1241590) takes it as the white off-hand
// swing does.
func TestMutilateOffHandTakesDualWieldSpecialization(t *testing.T) {
	const ap = 1000.0
	base := talentStringWithRank(t, zeroRogueTalents, "mutilate", 1)
	sim0, r0 := rogueSim(t, base, "combat_backstab_prebis")
	sim2, r2 := rogueSim(t, talentStringWithRank(t, base, "dual_wield_specialization", 2), "combat_backstab_prebis")
	without := r0.GetRogue().MutilateOHWeaponDamageForTest(sim0, ap)
	with := r2.GetRogue().MutilateOHWeaponDamageForTest(sim2, ap)
	if math.Abs(with/without-1.50) > 1e-9 {
		t.Errorf("Mutilate off-hand weapon damage with DWS 2/2 = %.3f x untalented, want 1.50", with/without)
	}
}
