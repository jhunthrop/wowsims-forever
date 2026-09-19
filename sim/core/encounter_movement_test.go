package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// An encounter with no movement block is exactly today's encounter: the
// feature is additive and must not cost an unmoving fight anything.
func TestEncounterWithoutMovementHasNone(t *testing.T) {
	enc := NewEncounter(&proto.Encounter{
		Duration: 180,
		Targets:  []*proto.Target{DefaultTargetProtoLvl60},
	})
	if enc.Movement != nil {
		t.Errorf("Encounter.Movement = %+v, want nil for an encounter that sets none", enc.Movement)
	}
}

func TestEncounterParsesItsMovementPattern(t *testing.T) {
	enc := NewEncounter(&proto.Encounter{
		Duration: 180,
		Targets:  []*proto.Target{DefaultTargetProtoLvl60},
		Movement: &proto.MovementPattern{IntervalSeconds: 45, DurationSeconds: 5},
	})
	if enc.Movement == nil {
		t.Fatal("Encounter.Movement = nil, want a pattern")
	}
	if enc.Movement.Interval != 45*time.Second {
		t.Errorf("Interval = %s, want 45s", enc.Movement.Interval)
	}
	if enc.Movement.Duration != 5*time.Second {
		t.Errorf("Duration = %s, want 5s", enc.Movement.Duration)
	}
	if enc.Movement.CastingOnly {
		t.Error("CastingOnly = true, want false")
	}
}

// A zero interval or a zero duration is not a movement pattern, it is an
// unset one. Failing open here would schedule an infinite loop of
// zero-length windows.
func TestEncounterIgnoresADegenerateMovementPattern(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pattern *proto.MovementPattern
	}{
		{"zero interval", &proto.MovementPattern{IntervalSeconds: 0, DurationSeconds: 5}},
		{"zero duration", &proto.MovementPattern{IntervalSeconds: 45, DurationSeconds: 0}},
		{"negative interval", &proto.MovementPattern{IntervalSeconds: -1, DurationSeconds: 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			enc := NewEncounter(&proto.Encounter{
				Duration: 180,
				Targets:  []*proto.Target{DefaultTargetProtoLvl60},
				Movement: tc.pattern,
			})
			if enc.Movement != nil {
				t.Errorf("Encounter.Movement = %+v, want nil", enc.Movement)
			}
		})
	}
}

// The away window puts the unit out of melee and stops it casting; when
// the window closes the unit is back at the distance it configured.
func TestAwayMovementWindowMovesOutAndBack(t *testing.T) {
	sim, unit := movementSim(t, &proto.MovementPattern{IntervalSeconds: 20, DurationSeconds: 5})
	startDistance := unit.DistanceFromTarget

	unit.startMovementWindow(sim, sim.Encounter.Movement)
	if !unit.IsMoving() {
		t.Error("unit.IsMoving() = false inside an away window, want true")
	}
	if !unit.IsCastingBlocked() {
		t.Error("unit.IsCastingBlocked() = false inside an away window, want true")
	}
	if unit.DistanceFromTarget != EncounterMovementDistance {
		t.Errorf("DistanceFromTarget = %v inside the window, want %v", unit.DistanceFromTarget, EncounterMovementDistance)
	}

	advanceSimTo(sim, 5*time.Second)
	if unit.IsMoving() {
		t.Error("unit.IsMoving() = true after the window closed, want false")
	}
	if unit.DistanceFromTarget != startDistance {
		t.Errorf("DistanceFromTarget = %v after the window, want the configured %v", unit.DistanceFromTarget, startDistance)
	}
}

// casting_only is the other half of the contract: casting stops, the unit
// never leaves melee, so autos keep swinging.
func TestCastingOnlyWindowBlocksCastingWithoutMoving(t *testing.T) {
	sim, unit := movementSim(t, &proto.MovementPattern{IntervalSeconds: 20, DurationSeconds: 5, CastingOnly: true})
	startDistance := unit.DistanceFromTarget

	unit.startMovementWindow(sim, sim.Encounter.Movement)
	if unit.IsMoving() {
		t.Error("unit.IsMoving() = true in a casting-only window, want false")
	}
	if !unit.IsCastingBlocked() {
		t.Error("unit.IsCastingBlocked() = false in a casting-only window, want true")
	}
	if unit.DistanceFromTarget != startDistance {
		t.Errorf("DistanceFromTarget = %v in a casting-only window, want the unchanged %v", unit.DistanceFromTarget, startDistance)
	}

	advanceSimTo(sim, 5*time.Second)
	if unit.IsCastingBlocked() {
		t.Error("unit.IsCastingBlocked() = true after the window closed, want false")
	}
}

// movementSim builds a one-player, one-target simulation with the given
// movement pattern, reset and ready to step.
func movementSim(t *testing.T, pattern *proto.MovementPattern) (*Simulation, *Unit) {
	t.Helper()
	sim := NewSim(&proto.RaidSimRequest{
		Raid: SinglePlayerRaidProto(&proto.Player{
			Name:               "Movement Test",
			Race:               proto.Race_RaceOrc,
			Class:              proto.Class_ClassShaman,
			Spec:               &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment:          &proto.EquipmentSpec{},
			DistanceFromTarget: 5,
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration: 180,
			Targets:  []*proto.Target{DefaultTargetProtoLvl60},
			Movement: pattern,
		},
		SimOptions: &proto.SimOptions{Iterations: 1, IsTest: true, RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.reset()
	return sim, &sim.Raid.Parties[0].Players[0].GetCharacter().Unit
}

// advanceSimTo steps the event loop until the clock reaches `at`, so a
// delayed action scheduled for exactly `at` actually fires. It stops as
// soon as CurrentTime reaches `at` rather than stepping past it: this
// test's Movement.Interval (20s) is far larger than the window duration
// (5s), and nothing else is scheduled in between for this minimal
// fixture (no mana bar, no auto attacks, no APL rotation), so a step
// past `at` would run straight into the encounter's *next* window and
// make the assertions racy against a real event rather than the
// boundary one.
func advanceSimTo(sim *Simulation, at time.Duration) {
	for sim.CurrentTime < at {
		if finished := sim.Step(); finished {
			return
		}
	}
}
