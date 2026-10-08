package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

func TestConeOfColdRegistersByLevel(t *testing.T) {
	cases := []struct {
		level int32
		want  []int
	}{
		{25, nil},
		{26, []int{1}},
		{42, []int{1, 2, 3}},
		{60, []int{1, 2, 3, 4, 5}},
	}
	for _, tc := range cases {
		_, mage := newMageAtLevel(t, tc.level, ForeverFrostTalents)
		var got []int
		for rank, spell := range mage.ConeOfCold {
			if spell != nil {
				got = append(got, rank)
			}
		}
		if len(got) != len(tc.want) {
			t.Errorf("level %d: registered ranks %v, want %v", tc.level, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("level %d: registered ranks %v, want %v", tc.level, got, tc.want)
			}
		}
	}
}

// The client's cooldown is the 10 s category cooldown on every rank, and
// the spell is instant.
func TestConeOfColdIsInstantWithTheClientCooldown(t *testing.T) {
	_, mage := newMageAtLevel(t, 60, ForeverFrostTalents)
	cone := mage.ConeOfCold[ConeOfColdRanks]
	if cone.CD.Duration != 10*time.Second {
		t.Errorf("cooldown %v, want 10s", cone.CD.Duration)
	}
	if cone.DefaultCast.CastTime != 0 {
		t.Errorf("cast time %v, want instant", cone.DefaultCast.CastTime)
	}
}

func TestConeOfColdIsAChillEffectAndGrantsFingersOfFrost(t *testing.T) {
	if !grantedWithin(t, func(m *Mage) *core.Spell { return m.ConeOfCold[ConeOfColdRanks] }, 40) {
		t.Error("40 Cone of Cold casts never granted Fingers of Frost")
	}
}

func TestConeOfColdIsBinaryLikeFrostbolt(t *testing.T) {
	_, mage := newMageAtLevel(t, 60, ForeverFrostTalents)
	if !mage.ConeOfCold[ConeOfColdRanks].Flags.Matches(core.SpellFlagBinary) {
		t.Error("Cone of Cold has the same damage-plus-snare structure as Frostbolt, which is binary")
	}
}

func newFrostMageSimAgainstTargets(t *testing.T, targets []*proto.Target) (*Mage, *core.Simulation) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      ForeverFrostTalents,
			DistanceFromTarget: 5,
		},
		PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: targets},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	return sim.Raid.Parties[0].Players[0].(MageAgent).GetMage(), sim
}

func TestConeOfColdHitsEveryTarget(t *testing.T) {
	built, sim := newFrostMageSimAgainstTargets(t, []*proto.Target{bossTarget(), bossTarget()})
	cone := built.ConeOfCold[ConeOfColdRanks]
	for i := 0; i < 20; i++ {
		cone.ApplyEffects(sim, sim.Encounter.TargetUnits[0], cone)
	}
	for i, unit := range sim.Encounter.TargetUnits {
		if cone.SpellMetrics[unit.UnitIndex].TotalDamage <= 0 {
			t.Errorf("target %d took no Cone of Cold damage", i)
		}
	}
}

func coneDamage(t *testing.T, talents string, casts int) float64 {
	t.Helper()
	built, sim, boss := newFrostMageSimWithTalents(t, bossTarget(), talents)
	cone := built.ConeOfCold[ConeOfColdRanks]
	for i := 0; i < casts; i++ {
		cone.ApplyEffects(sim, boss, cone)
	}
	return cone.SpellMetrics[boss.UnitIndex].TotalDamage
}

// Improved Cone of Cold is "+12/23/35% damage dealt by your Cone of Cold".
func TestImprovedConeOfColdRaisesConeDamage(t *testing.T) {
	const casts = 600
	none := coneDamage(t, talentStringWithRank(t, ForeverFrostTalents, "improved_cone_of_cold", 0), casts)
	full := coneDamage(t, talentStringWithRank(t, ForeverFrostTalents, "improved_cone_of_cold", 3), casts)
	ratio := full / none
	if ratio < 1.28 || ratio > 1.42 {
		t.Errorf("Improved Cone of Cold 3/3 changed Cone damage by x%.3f, want about 1.35", ratio)
	}
}
