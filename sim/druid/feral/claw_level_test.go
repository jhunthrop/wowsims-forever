package feral

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// TestClawLevel60HasMaxRankAndDealsDamage rebuilds the case that broke:
// registerClawSpell hardcoded ActionID{SpellID: 9850} (Claw's rank 5,
// learned at 58) with no per-rank loop, unlike shred.go and rake.go in the
// same package. The site's rank rewrite resolves a rotation's Claw to the
// id the caster's level has actually learned, so any Feral druid below
// level 58 got a spell id this engine never registered and cast nothing.
// A level-60 druid must still register Claw at its max rank (9850, per
// data/builds/1.60.1.70009's spellranks.json) and casting it must actually
// deal melee damage.
func TestClawLevel60HasMaxRankAndDealsDamage(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassDruid,
			Race:               proto.Race_RaceTauren,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{}, // no gear: this fork's item database isn't generated in this test environment.
			Buffs:              core.FullBuffs.Player,
			TalentsString:      P1Talents,
			DistanceFromTarget: 5,
		},
		PlayerOptionsMonoCat,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	built, ok := sim.Raid.Parties[0].Players[0].(*FeralDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *FeralDruid")
	}

	if built.Claw == nil {
		t.Fatal("level-60 Feral druid has no Claw registered")
	}
	if got, want := built.Claw.ActionID.SpellID, int32(9850); got != want {
		t.Errorf("Claw spell ID = %d, want max rank %d", got, want)
	}
	if got, want := built.Claw.Rank, 5; got != want {
		t.Errorf("Claw rank = %d, want %d", got, want)
	}

	target := sim.Encounter.TargetUnits[0]

	// Drive the spell's own ApplyEffects directly (the real code a Cast
	// would run) rather than going through Cast, so the assertion isn't
	// gated on GCD/energy/form-shift bookkeeping that's out of scope here.
	built.Claw.ApplyEffects(sim, target, built.Claw.Spell)

	metrics := built.Claw.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Claw outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Claw dealt %v damage, want > 0", metrics.TotalDamage)
	}
}

// TestClawLevel20HasRankOne is the brief's other case: a level-20 Feral
// druid (the level spellranks.json gives rank 1, id 1082, and nothing
// else) must have Claw registered at rank 1 - not nil, and not some
// higher rank the character hasn't learned - and casting it must land.
func TestClawLevel20HasRankOne(t *testing.T) {
	const wantLevel = 20

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassDruid,
			Race:               proto.Race_RaceTauren,
			Level:              wantLevel,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      P1Talents,
			DistanceFromTarget: 5,
		},
		PlayerOptionsMonoCat,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	// core.NewDefaultTarget() is a fixed level-63 raid-boss dummy: fine for
	// the level-60 case above, but a level-20 attacker's defense-skill gap
	// against it makes every special nearly certain to be dodged (verified:
	// 20 casts against it, 19 dodges), which would make this test assert
	// nothing about whether Claw *lands* at rank 1. Use a level-matched
	// dummy instead, the way an actual level-20 Claw cast would land.
	target := &proto.Target{
		Level: wantLevel,
		Stats: stats.Stats{
			stats.Armor: 100,
		}.ToFloatArray(),
	}

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{target},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	built, ok := sim.Raid.Parties[0].Players[0].(*FeralDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *FeralDruid")
	}
	if built.Level != wantLevel {
		t.Fatalf("druid.Level = %d, want %d", built.Level, wantLevel)
	}

	if built.Claw == nil {
		t.Fatal("level-20 Feral druid has no Claw registered")
	}
	if got, want := built.Claw.ActionID.SpellID, int32(1082); got != want {
		t.Errorf("Claw spell ID = %d, want rank 1 %d", got, want)
	}
	if got, want := built.Claw.Rank, 1; got != want {
		t.Errorf("Claw rank = %d, want %d", got, want)
	}

	tgt := sim.Encounter.TargetUnits[0]

	// Several casts against a level-matched dummy, rather than one, so a
	// single unlucky dodge/miss roll can't make this test flaky.
	const attempts = 10
	for i := 0; i < attempts; i++ {
		built.Claw.ApplyEffects(sim, tgt, built.Claw.Spell)
	}

	metrics := built.Claw.SpellMetrics[tgt.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Claw landed 0 of %d casts (misses=%d dodges=%d parries=%d)", attempts, metrics.Misses, metrics.Dodges, metrics.Parries)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Claw dealt %v damage, want > 0", metrics.TotalDamage)
	}
}
