package mage

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// TestFrostNovaFreezesAFreezableTargetAndIceLanceBenefits rebuilds the
// case ice_lance.go used to leave stubbed: isTargetFrozen/frozenAuras
// always read false because no ability in this package ever registered
// a Freeze. A level-60 Frost mage's Frost Nova must root a freezable
// (sub-cap level) target, and Ice Lance's x3 Frozen bonus
// (iceLanceFrozenMultiplier) must actually show up once it does.
func TestFrostNovaFreezesAFreezableTargetAndIceLanceBenefits(t *testing.T) {
	built, sim, target := newFrostMageSim(t, &proto.Target{
		Level: 60, // <= core.CharacterMaxLevel: freezable.
		Stats: core.DefaultTargetProtoLvl60.Stats,
	})

	if len(built.FrostNova) <= FrostNovaRanks || built.FrostNova[FrostNovaRanks] == nil {
		t.Fatalf("level-60 mage has no top-rank Frost Nova registered")
	}
	if len(built.IceLance) <= IceLanceRanks || built.IceLance[IceLanceRanks] == nil {
		t.Fatalf("level-60 Frost mage (ForeverFrostTalents takes Ice Lance) has no top-rank Ice Lance registered")
	}

	if built.isTargetFrozen(target) {
		t.Fatal("target is Frozen before Frost Nova was ever cast")
	}

	frostNova := built.FrostNova[FrostNovaRanks]
	frostNova.ApplyEffects(sim, target, frostNova)

	if !built.isTargetFrozen(target) {
		t.Fatal("target is not Frozen after a landed Frost Nova")
	}

	// Ice Lance's x3 Frozen multiplier (ice_lance.go) is a plain scalar
	// on base damage, so it holds in expectation regardless of the
	// hit/crit roll on any one cast; averaging enough casts with a fixed
	// seed makes the comparison deterministic without asserting an exact
	// ratio.
	const iterations = 200
	iceLance := built.IceLance[IceLanceRanks]
	for i := 0; i < iterations; i++ {
		iceLance.ApplyEffects(sim, target, iceLance)
	}
	waitForOutcomes(sim)
	frozenDamage := iceLance.SpellMetrics[target.UnitIndex].TotalDamage
	if frozenDamage <= 0 {
		t.Fatalf("Ice Lance dealt no damage against a Frozen target over %d casts", iterations)
	}

	// A second, never-Frozen target for the baseline comparison.
	baseline, sim2, plainTarget := newFrostMageSim(t, &proto.Target{
		Level: 60,
		Stats: core.DefaultTargetProtoLvl60.Stats,
	})
	if baseline.isTargetFrozen(plainTarget) {
		t.Fatal("baseline target is unexpectedly Frozen")
	}
	iceLance2 := baseline.IceLance[IceLanceRanks]
	for i := 0; i < iterations; i++ {
		iceLance2.ApplyEffects(sim2, plainTarget, iceLance2)
	}
	waitForOutcomes(sim2)
	unfrozenDamage := iceLance2.SpellMetrics[plainTarget.UnitIndex].TotalDamage

	// True expected ratio is iceLanceFrozenMultiplier (3.0); a 1.5x
	// floor over 200 samples is comfortably outside sampling noise while
	// still failing hard if the Frozen bonus stops applying.
	if frozenDamage < unfrozenDamage*1.5 {
		t.Errorf("Ice Lance against a Frozen target averaged %v total damage over %d casts, want at least 1.5x the unfrozen baseline %v",
			frozenDamage, iterations, unfrozenDamage)
	}
}

// TestFrostNovaCannotFreezeABoss is the other half of canFreeze
// (frost_nova.go): every raid boss in this sim is built above
// core.CharacterMaxLevel (core/target.go's defaultRaidBossLevel is
// CharacterMaxLevel+3), so Frost Nova's root - and so Ice Lance and
// Shatter's Frozen bonus - must never land on one.
func TestFrostNovaCannotFreezeABoss(t *testing.T) {
	built, sim, boss := newFrostMageSim(t, core.DefaultTargetProtoLvl60) // level 63

	frostNova := built.FrostNova[FrostNovaRanks]
	for i := 0; i < 20; i++ {
		frostNova.ApplyEffects(sim, boss, frostNova)
	}

	if built.isTargetFrozen(boss) {
		t.Fatal("a level-63 boss was Frozen by Frost Nova; canFreeze should have refused it")
	}
}

// newFrostMageSim builds a level-60 Frost mage (ForeverFrostTalents, so
// Ice Lance and Missile Barrage are both talented) against a single
// custom target, with no gear - this fork's item database isn't
// generated in this test environment (see pet_level_test.go) - and
// returns the built mage, a reset+pre-pulled sim, and the target unit.
func newFrostMageSim(t *testing.T, target *proto.Target) (*Mage, *core.Simulation, *core.Unit) {
	t.Helper()
	return newFrostMageSimWithTalents(t, target, ForeverFrostTalents)
}

// newFrostMageSimWithTalents is newFrostMageSim with the talent string
// chosen by the test, for the rank-dependent talents.
func newFrostMageSimWithTalents(t *testing.T, target *proto.Target, talents string) (*Mage, *core.Simulation, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talents,
			DistanceFromTarget: 5,
		},
		PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

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

	agent, ok := sim.Raid.Parties[0].Players[0].(MageAgent)
	if !ok {
		t.Fatal("the raid's first player is not a mage agent")
	}
	built := agent.GetMage()

	return built, sim, sim.Encounter.TargetUnits[0]
}

// waitForOutcomes steps sim forward exactly one second of sim time -
// comfortably longer than Ice Lance's or Arcane Blast's missile travel
// time (well under 1s at this test's DistanceFromTarget) so every
// spell.WaitTravelTime pending action queued so far fires, the way
// aimed_shot_test.go's TestAimedShotLevel60HasMaxRankAndDealsDamage
// waits out a single one - but short enough to stay well inside Arcane
// Blast's 8s stacking buff, unlike stepping until combat ends.
func waitForOutcomes(sim *core.Simulation) {
	deadline := sim.CurrentTime + time.Second
	for i := 0; i < 10000 && sim.CurrentTime < deadline; i++ {
		if sim.Step() {
			return
		}
	}
}
