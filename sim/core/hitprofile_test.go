package core

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// A level-60 dual wielder against a level-63 boss at 300 skill: 8% base
// miss, a 1% suppression dead zone, specials capped at 9% hit and white
// swings (19% dual wield penalty) at 28%.
var dualWielder = HitProfile{Physical: true, Suppression: 1, SpecialCap: 9, WhiteCap: 28, DualWielding: true}

func profileAt(base HitProfile, hit float64) HitProfile {
	base.Hit = hit
	return base
}

func TestHitStepWindowIsTwoSidedInsideTheLiveRange(t *testing.T) {
	low, high, ok := hitStepWindow(profileAt(dualWielder, 5))
	if !ok || low != -1 || high != 1 {
		t.Fatalf("window at 5%% hit = (%v, %v, %v), want (-1, 1, true)", low, high, ok)
	}
}

func TestHitStepWindowIsOneSidedWhereTheLowSideIsFloored(t *testing.T) {
	// At 0% the dead zone (0 to 1%) is flat, so the live slope starts at
	// the suppression edge: from +1 (the edge) to +2.
	low, high, ok := hitStepWindow(profileAt(dualWielder, 0))
	if !ok || low != 1 || high != 2 {
		t.Fatalf("window at 0%% hit = (%v, %v, %v), want (1, 2, true)", low, high, ok)
	}
	// At 1.5% the floor is half a point away: step from the edge to +1.
	low, high, ok = hitStepWindow(profileAt(dualWielder, 1.5))
	if !ok || low != -0.5 || high != 1 {
		t.Fatalf("window at 1.5%% hit = (%v, %v, %v), want (-0.5, 1, true)", low, high, ok)
	}
}

func TestHitStepWindowIsOneSidedWhereTheHighSideIsCapped(t *testing.T) {
	low, high, ok := hitStepWindow(profileAt(dualWielder, 28))
	if !ok || low != -1 || high != 0 {
		t.Fatalf("window at the white cap = (%v, %v, %v), want (-1, 0, true)", low, high, ok)
	}
}

func TestHitStepWindowSkipsACapturedStat(t *testing.T) {
	if _, _, ok := hitStepWindow(profileAt(dualWielder, 30)); ok {
		t.Fatal("a character past the white cap must not be swept: hit is worth nothing there")
	}
}

func TestHitStepWindowIsSymmetricForACaster(t *testing.T) {
	low, high, ok := hitStepWindow(HitProfile{Hit: 3})
	if !ok || low != -1 || high != 1 {
		t.Fatalf("caster window = (%v, %v, %v), want (-1, 1, true)", low, high, ok)
	}
}

func TestHitDistanceToCaps(t *testing.T) {
	p := profileAt(dualWielder, 4)
	if got := p.ToSpecialCap(); got != 5 {
		t.Errorf("ToSpecialCap = %v, want 5", got)
	}
	if got := p.ToWhiteCap(); got != 24 {
		t.Errorf("ToWhiteCap = %v, want 24", got)
	}
	if got := profileAt(dualWielder, 12).ToSpecialCap(); got != 0 {
		t.Errorf("ToSpecialCap past the cap = %v, want 0", got)
	}
}

// A window that is not the symmetric default is read as the DPS
// difference between its two simulations over their distance.
func TestComputeStatWeightsReadsAnAsymmetricWindowAsASlope(t *testing.T) {
	values := func(v float64, n int) *proto.DistributionMetrics {
		m := &proto.DistributionMetrics{Avg: v}
		for i := 0; i < n; i++ {
			m.AllValues = append(m.AllValues, v)
		}
		return m
	}
	player := func(dps float64) *proto.RaidSimResult {
		return &proto.RaidSimResult{RaidMetrics: &proto.RaidMetrics{Parties: []*proto.PartyMetrics{{Players: []*proto.UnitMetrics{{
			Dps: values(dps, 4), Hps: values(0, 4), Threat: values(0, 4), Dtps: values(0, 4), Tmi: values(0, 4),
		}}}}}}
	}
	ref := stats.UnitStatFromStat(stats.AttackPower)
	hit := stats.UnitStatFromStat(stats.Hit)
	// Baseline 100 DPS, the window is [-1, 0]: 98 at -1 and 100 at the baseline.
	// An unrelated reference stat keeps the result well formed.
	result := computeStatWeights(&proto.StatWeightsCalcRequest{
		BaseResult:      player(100),
		EpReferenceStat: proto.Stat_StatAttackPower,
		StatSimResults: []*proto.StatWeightsStatResultData{
			{
				StatData:   &proto.StatWeightsStatData{UnitStat: int32(ref), ModLow: -1, ModHigh: 1},
				ResultLow:  player(99),
				ResultHigh: player(101),
			},
			{
				StatData:   &proto.StatWeightsStatData{UnitStat: int32(hit), ModLow: -1, ModHigh: 0},
				ResultLow:  player(98),
				ResultHigh: player(100),
			},
		},
	})
	got := result.Dps.Weights.Stats[stats.Hit]
	if math.Abs(got-2) > 1e-9 {
		t.Fatalf("hit weight over the window [-1, 0] = %v, want 2 DPS per point", got)
	}
}

func TestExpertiseDistanceToCaps(t *testing.T) {
	p := HitProfile{Melee: true, Expertise: 2, DodgeChance: 6.5, ParryChance: 14}
	if got := p.ToDodgeCap(); got != 4.5 {
		t.Errorf("ToDodgeCap = %v, want 4.5", got)
	}
	if got := p.ToParryCap(); got != 12 {
		t.Errorf("ToParryCap = %v, want 12", got)
	}
	p.Expertise = 20
	if p.ToDodgeCap() != 0 || p.ToParryCap() != 0 {
		t.Errorf("past both chances the distances must floor at 0, got %v and %v", p.ToDodgeCap(), p.ToParryCap())
	}
}
