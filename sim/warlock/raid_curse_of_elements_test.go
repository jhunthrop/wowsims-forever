package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// TestRaidCurseOfElementsDebuffIsTheTopRankOfTheForeverCurse checks the shared
// raid-debuff aura (core.CurseOfElementsAura) against rank 4 of the client's
// curse (1311680): every magic school takes 10 percent more damage and 75
// resistance is removed; physical is untouched.
func TestRaidCurseOfElementsDebuffIsTheTopRankOfTheForeverCurse(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{Class: proto.Class_ClassWarlock, Race: proto.Race_RaceOrc, Level: 60, Equipment: &proto.EquipmentSpec{}},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{}}},
	)
	build := func(debuffs *proto.Debuffs) *core.Unit {
		raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, debuffs)
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       raid,
			Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
			SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
		}, simsignals.CreateSignals())
		sim.Reset()
		sim.PrePull()
		return sim.Encounter.TargetUnits[0]
	}
	bare := build(&proto.Debuffs{})
	cursed := build(&proto.Debuffs{CurseOfElements: true})

	for _, school := range curseOfTheElementsSchools {
		got := cursed.PseudoStats.SchoolDamageTakenMultiplier[school] / bare.PseudoStats.SchoolDamageTakenMultiplier[school]
		if math.Abs(got-1.10) > 1e-9 {
			t.Errorf("school %d: damage taken x%.4f, want x1.10", school, got)
		}
	}
	physical := stats.SchoolIndexPhysical
	if cursed.PseudoStats.SchoolDamageTakenMultiplier[physical] != bare.PseudoStats.SchoolDamageTakenMultiplier[physical] {
		t.Error("the curse changes physical damage taken")
	}
	for _, resistance := range []stats.Stat{stats.ArcaneResistance, stats.FireResistance, stats.FrostResistance, stats.NatureResistance, stats.ShadowResistance} {
		if got := bare.GetStat(resistance) - cursed.GetStat(resistance); got != 75 {
			t.Errorf("stat %v: resistance removed %.0f, want 75", resistance, got)
		}
	}
}
