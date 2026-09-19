package sim

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/mage"
	"github.com/wowsims/classic/sim/warrior"
)

// parityIterations is small on purpose: these are direction-of-travel
// checks over a three-minute fight, not a tuning fixture, and the whole
// file has to stay inside the fork's test budget.
const parityIterations = 30

// furyWarriorPlayer is the reference melee: rage, no cast times, so an
// away window costs it auto attacks and a casting-only window costs it
// nothing.
func furyWarriorPlayer() *proto.Player {
	return &proto.Player{
		Name:          "Fury Warrior",
		Race:          proto.Race_RaceOrc,
		Class:         proto.Class_ClassWarrior,
		Equipment:     core.GetGearSet("../ui/warrior/gear_sets", "phase_1").GearSet,
		TalentsString: warrior.ForeverFuryTalents,
		Rotation:      core.GetAplRotation("../ui/warrior/apls", "forever_fury").Rotation,
		Consumes:      &proto.Consumes{},
		Buffs:         &proto.IndividualBuffs{},
		Spec: &proto.Player_Warrior{Warrior: &proto.Warrior{
			Options: &proto.Warrior_Options{
				StartingRage: 50,
				Shout:        proto.WarriorShout_WarriorShoutBattle,
			},
		}},
		DistanceFromTarget: 5,
	}
}

// frostMagePlayer is the reference caster: mana and long cast times, so
// both window kinds cost it casts.
func frostMagePlayer() *proto.Player {
	return &proto.Player{
		Name:          "Frost Mage",
		Race:          proto.Race_RaceTroll,
		Class:         proto.Class_ClassMage,
		Equipment:     core.GetGearSet("../ui/mage/gear_sets", "p0.bis").GearSet,
		TalentsString: mage.ForeverFrostTalents,
		Rotation:      core.GetAplRotation("../ui/mage/apls", "forever_frost").Rotation,
		Consumes:      &proto.Consumes{},
		Buffs:         &proto.IndividualBuffs{},
		Spec: &proto.Player_Mage{Mage: &proto.Mage{
			Options: &proto.Mage_Options{Armor: proto.Mage_Options_MoltenArmor},
		}},
		DistanceFromTarget: 20,
	}
}

// parityEncounter is a plain three-minute Patchwerk with no duration
// variation, so two runs differ only by the feature under test.
func parityEncounter() *proto.Encounter {
	return &proto.Encounter{
		Duration: 180,
		Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
	}
}

// runParitySim runs one request and returns the raid's mean DPS.
func runParitySim(t *testing.T, player *proto.Player, encounter *proto.Encounter) float64 {
	t.Helper()
	result := core.RunSim(&proto.RaidSimRequest{
		Raid:      core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: encounter,
		SimOptions: &proto.SimOptions{
			Iterations: parityIterations,
			IsTest:     true,
			RandomSeed: 1,
		},
	}, nil, simsignals.CreateSignals())
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	return result.RaidMetrics.Dps.Avg
}

// Heavy movement (5s away every 20s) is a quarter of the fight spent out
// of melee. A fury warrior's DPS must fall, and it must fall by less than
// the whole quarter, because rage carries across the window.
func TestHeavyMovementCostsTheFuryWarrior(t *testing.T) {
	still := runParitySim(t, furyWarriorPlayer(), parityEncounter())

	moving := parityEncounter()
	moving.Movement = &proto.MovementPattern{IntervalSeconds: 20, DurationSeconds: 5}
	movingDps := runParitySim(t, furyWarriorPlayer(), moving)

	if movingDps >= still {
		t.Errorf("heavy movement DPS %.1f is not below the standing DPS %.1f", movingDps, still)
	}
	if movingDps < still*0.5 {
		t.Errorf("heavy movement DPS %.1f is below half the standing DPS %.1f; a quarter of the fight away should not halve it", movingDps, still)
	}
}

// A casting-only window costs a melee nothing: it never leaves melee and
// it has no cast times to interrupt.
func TestCastingOnlyMovementIsFreeForTheFuryWarrior(t *testing.T) {
	still := runParitySim(t, furyWarriorPlayer(), parityEncounter())

	interrupted := parityEncounter()
	interrupted.Movement = &proto.MovementPattern{IntervalSeconds: 20, DurationSeconds: 5, CastingOnly: true}
	interruptedDps := runParitySim(t, furyWarriorPlayer(), interrupted)

	if interruptedDps != still {
		t.Errorf("casting-only DPS %.4f differs from the standing DPS %.4f; a melee with no cast times should be untouched", interruptedDps, still)
	}
}

// Both window kinds cost a frost mage, because every window interrupts a
// Frostbolt and refuses the next one for its duration.
func TestBothMovementKindsCostTheFrostMage(t *testing.T) {
	still := runParitySim(t, frostMagePlayer(), parityEncounter())

	away := parityEncounter()
	away.Movement = &proto.MovementPattern{IntervalSeconds: 20, DurationSeconds: 5}
	awayDps := runParitySim(t, frostMagePlayer(), away)

	castingOnly := parityEncounter()
	castingOnly.Movement = &proto.MovementPattern{IntervalSeconds: 20, DurationSeconds: 5, CastingOnly: true}
	castingOnlyDps := runParitySim(t, frostMagePlayer(), castingOnly)

	if awayDps >= still {
		t.Errorf("away-movement DPS %.1f is not below the standing DPS %.1f", awayDps, still)
	}
	if castingOnlyDps >= still {
		t.Errorf("casting-only DPS %.1f is not below the standing DPS %.1f", castingOnlyDps, still)
	}
}

// Light movement (5s every 45s) costs less than heavy movement (5s every
// 20s). This is the ordering the two fight styles promise the player.
func TestLightMovementCostsLessThanHeavy(t *testing.T) {
	light := parityEncounter()
	light.Movement = &proto.MovementPattern{IntervalSeconds: 45, DurationSeconds: 5}
	heavy := parityEncounter()
	heavy.Movement = &proto.MovementPattern{IntervalSeconds: 20, DurationSeconds: 5}

	lightDps := runParitySim(t, frostMagePlayer(), light)
	heavyDps := runParitySim(t, frostMagePlayer(), heavy)

	if lightDps <= heavyDps {
		t.Errorf("light-movement DPS %.1f is not above heavy-movement DPS %.1f", lightDps, heavyDps)
	}
}
