package hunter

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// TestAimedShotLevel60HasMaxRankAndDealsDamage rebuilds the case that broke:
// registerAimedShotSpell's body was entirely commented out, so no hunter
// could ever cast Aimed Shot. A level-60 hunter must have the spell
// registered at its max rank (20904, per data/builds/1.60.1.70009's
// spellranks.json), and casting it must actually deal ranged damage - not
// just exist as a *core.Spell wired to nothing.
func TestAimedShotLevel60HasMaxRankAndDealsDamage(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{}, // no gear: this fork's item database isn't generated in this test environment (see pet_level_test.go).
			Buffs:              core.FullBuffs.Player,
			DistanceFromTarget: 25, // outside core.MinRangedAttackDistance, so Aimed Shot's ExtraCastCondition allows it.
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(HunterAgent)
	if !ok {
		t.Fatal("the raid's first player is not a hunter agent")
	}
	built := agent.GetHunter()

	if built.AimedShot == nil {
		t.Fatal("level-60 hunter has no Aimed Shot registered")
	}
	if got, want := built.AimedShot.ActionID.SpellID, int32(20904); got != want {
		t.Errorf("Aimed Shot spell ID = %d, want max rank %d", got, want)
	}
	if got, want := built.AimedShot.Rank, 6; got != want {
		t.Errorf("Aimed Shot rank = %d, want %d", got, want)
	}

	target := sim.Encounter.TargetUnits[0]

	// Drive the spell's own ApplyEffects directly (the real code a Cast
	// would run) rather than going through Cast, so the assertion isn't
	// gated on GCD/mana/cooldown bookkeeping that's out of scope here.
	built.AimedShot.ApplyEffects(sim, target, built.AimedShot)

	// The hit/crit outcome is rolled synchronously inside ApplyEffects;
	// only the damage total is deferred behind the missile's travel time
	// (spell.WaitTravelTime), so step the sim until that pending action
	// fires.
	for i := 0; i < 200; i++ {
		if built.AimedShot.SpellMetrics[target.UnitIndex].TotalDamage > 0 {
			break
		}
		if done := sim.Step(); done {
			break
		}
	}

	metrics := built.AimedShot.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Aimed Shot outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Aimed Shot dealt %v damage, want > 0", metrics.TotalDamage)
	}
}

// TestAimedShotLevel20HasRankOne is the brief's second case: a level-20
// hunter (the level spellranks.json gives rank 1, id 19434) must have
// Aimed Shot registered at rank 1, not nil and not some higher rank.
func TestAimedShotLevel20HasRankOne(t *testing.T) {
	const wantLevel = 20

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              wantLevel,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)

	built, ok := env.Raid.Parties[0].Players[0].(*Hunter)
	if !ok {
		t.Fatal("player 0 did not build as a *Hunter")
	}
	if built.Level != wantLevel {
		t.Fatalf("hunter.Level = %d, want %d", built.Level, wantLevel)
	}

	if built.AimedShot == nil {
		t.Fatal("level-20 hunter has no Aimed Shot registered")
	}
	if got, want := built.AimedShot.ActionID.SpellID, int32(19434); got != want {
		t.Errorf("Aimed Shot spell ID = %d, want rank 1 %d", got, want)
	}
	if got, want := built.AimedShot.Rank, 1; got != want {
		t.Errorf("Aimed Shot rank = %d, want %d", got, want)
	}
}
