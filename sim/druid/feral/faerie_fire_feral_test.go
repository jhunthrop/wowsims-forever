package feral

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
)

// newUndebuffedFeralSim is a feral druid at level against a target with no
// raid debuffs, so the druid's own Faerie Fire is the only armor reduction.
func newUndebuffedFeralSim(t *testing.T, level int32) (*FeralDruid, *core.Simulation, *core.Unit) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassDruid,
			Race:               proto.Race_RaceTauren,
			Level:              level,
			Equipment:          &proto.EquipmentSpec{},
			TalentsString:      P1Talents,
			DistanceFromTarget: 5,
		},
		PlayerOptionsMonoCat,
	)
	raid := core.SinglePlayerRaidProto(player, nil, nil, nil)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{{Level: level, Stats: stats.Stats{stats.Armor: 2000}.ToFloatArray()}},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	built, ok := sim.Raid.Parties[0].Players[0].(*FeralDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *FeralDruid")
	}
	return built, sim, sim.Encounter.TargetUnits[0]
}

// Forever lets a druid cast Faerie Fire in Cat Form and Bear Form. Cat Form
// (Passive) 3025 and Bear Form (Passive) 1178 take 100% off its cost (aura
// 108, SpellModOp 14, mask 0x400) and add 6000 ms to its cooldown (aura 107,
// op 11), and Cat Form takes 500 ms off its global cooldown (op 21); Blizzard's
// 15 October notes speak of "a free Faerie Fire cast while using Cat Form,
// Bear Form, or Dire Bear Form", and Warcraft Tavern's Forever feral guide
// keeps it up in both rotations. The rank follows the level: at 30 it is
// rank 2 (778), 285 armor (effect 0, aura 674).
func TestFaerieFireIsAFreeSixSecondCastInCatForm(t *testing.T) {
	built, sim, target := newUndebuffedFeralSim(t, 30)
	ff := built.FaerieFireFeral
	if ff == nil {
		t.Fatal("a level-30 druid has no Faerie Fire for its feral forms")
	}
	if got, want := ff.ActionID, (core.ActionID{SpellID: 778, Tag: 1}); got != want {
		t.Errorf("feral Faerie Fire is %v at 30, want rank 2 (%v)", got, want)
	}
	if !built.InForm(druid.Cat) {
		built.CatFormAura.Activate(sim)
	}
	if !ff.CanCast(sim, target) {
		t.Fatal("Faerie Fire cannot be cast in Cat Form")
	}
	armorBefore := target.GetStat(stats.Armor)
	manaBefore := built.CurrentMana()
	ff.Cast(sim, target)

	if got := built.CurrentMana(); got != manaBefore {
		t.Errorf("Faerie Fire in Cat Form cost %.0f mana, want none", manaBefore-got)
	}
	if got := armorBefore - target.GetStat(stats.Armor); got != 285 {
		t.Errorf("rank 2 took %.0f armor off the target, want 285", got)
	}
	if got := ff.CD.TimeToReady(sim); got != 6*time.Second {
		t.Errorf("Faerie Fire's cooldown in Cat Form is %v, want 6s", got)
	}
	if got := built.GCD.TimeToReady(sim); got != time.Second {
		t.Errorf("Faerie Fire's global cooldown in Cat Form is %v, want 1s", got)
	}
}
