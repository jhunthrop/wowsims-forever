package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Dummy mode zeroes every execute proportion, whatever the request asked
// for: a dummy never drops below full health.
func TestDummyModeHasNoExecuteWindow(t *testing.T) {
	enc := NewEncounter(&proto.Encounter{
		Duration:             180,
		Targets:              []*proto.Target{DefaultTargetProtoLvl60},
		ExecuteProportion_20: 0.25,
		ExecuteProportion_25: 0.25,
		ExecuteProportion_35: 0.35,
		TargetDummy:          true,
	})
	if enc.Dummy != true {
		t.Error("Encounter.Dummy = false, want true")
	}
	if enc.ExecuteProportion_20 != 0 || enc.ExecuteProportion_25 != 0 || enc.ExecuteProportion_35 != 0 {
		t.Errorf("execute proportions = %v/%v/%v, want all zero on a dummy",
			enc.ExecuteProportion_20, enc.ExecuteProportion_25, enc.ExecuteProportion_35)
	}
}

// Without the flag nothing changes: the feature is additive.
func TestNonDummyKeepsItsExecuteWindow(t *testing.T) {
	enc := NewEncounter(&proto.Encounter{
		Duration:             180,
		Targets:              []*proto.Target{DefaultTargetProtoLvl60},
		ExecuteProportion_20: 0.25,
	})
	if enc.Dummy {
		t.Error("Encounter.Dummy = true, want false")
	}
	if enc.ExecuteProportion_20 != 0.25 {
		t.Errorf("ExecuteProportion_20 = %v, want 0.25", enc.ExecuteProportion_20)
	}
}

// Armor on a dummy is pinned to what the request configured: Sunder,
// Expose and Faerie Fire all lower stats.Armor dynamically, and on a
// dummy none of them may move the number the damage formula reads.
func TestDummyArmorIgnoresReduction(t *testing.T) {
	unit := &Unit{
		Type:         EnemyUnit,
		PseudoStats:  stats.NewPseudoStats(),
		initialStats: stats.Stats{stats.Armor: 3000},
	}
	unit.stats = unit.initialStats

	unit.stats[stats.Armor] = 1500
	if got := unit.Armor(); got != 1500 {
		t.Errorf("Armor() on a normal target = %v, want the reduced 1500", got)
	}

	unit.PseudoStats.ArmorReductionDisabled = true
	if got := unit.Armor(); got != 3000 {
		t.Errorf("Armor() on a dummy = %v, want the unreduced 3000", got)
	}
}

// The multiplier still applies on a dummy: it is gear and talents on the
// attacker's side of the table, not a debuff on the target.
func TestDummyArmorStillHonoursTheMultiplier(t *testing.T) {
	unit := &Unit{
		Type:         EnemyUnit,
		PseudoStats:  stats.NewPseudoStats(),
		initialStats: stats.Stats{stats.Armor: 3000},
	}
	unit.stats = unit.initialStats
	unit.PseudoStats.ArmorReductionDisabled = true
	unit.PseudoStats.ArmorMultiplier = 0.5

	if got := unit.Armor(); got != 1500 {
		t.Errorf("Armor() = %v, want 1500", got)
	}
}

// The raid debuff panel models other raiders, and a dummy has none, so
// none of it is applied and the target's armor is untouched by it.
func TestDummyModeSkipsTheRaidDebuffPanel(t *testing.T) {
	withDebuffs := dummyEnv(t, false, &proto.Debuffs{SunderArmor: true})
	onDummy := dummyEnv(t, true, &proto.Debuffs{SunderArmor: true})

	if !withDebuffs.Encounter.Targets[0].HasAura("Sunder Armor") {
		t.Fatal("a normal target has no Sunder Armor aura registered; the fixture is wrong, not the feature")
	}
	if onDummy.Encounter.Targets[0].HasAura("Sunder Armor") {
		t.Error("a dummy target has a Sunder Armor aura from the raid debuff panel, want none")
	}
}

// The positive half of switch 1: a debuff applied the way a player's own
// rotation applies one (not through the raid panel) must still register
// AND still take effect on a dummy target. Switch 3 (the armor gate)
// separately pins Armor() to its initial value regardless, so this test
// checks the aura and the stat it mutates directly, proving switch 1
// isn't accidentally gated by env.Encounter.Dummy the same way the raid
// panel is — a regression that would route the player's own debuffs
// through that gate would pass every other test in this file but fail
// here.
func TestDummyModePlayerDebuffStillLands(t *testing.T) {
	raidProto := SinglePlayerRaidProto(&proto.Player{
		Name:      "Dummy Test",
		Race:      proto.Race_RaceOrc,
		Class:     proto.Class_ClassShaman,
		Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
		Equipment: &proto.EquipmentSpec{},
	}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	encounterProto := &proto.Encounter{
		Duration:    180,
		Targets:     []*proto.Target{DefaultTargetProtoLvl60},
		TargetDummy: true,
	}

	// Build the environment in its normal three phases by hand (instead of
	// through NewEnvironment/dummyEnv) so the aura can be registered
	// between initialize and finalize — exactly when a real spec spell's
	// constructor registers its aura (auraTracker.registerAura panics on
	// any registration after finalize). Activating it and adding a stack
	// still happens after finalize, same as a real cast during combat.
	env := &Environment{State: Created}
	env.construct(raidProto, encounterProto)
	raidStats := env.initialize(raidProto, encounterProto)
	target := &env.Encounter.Targets[0].Unit

	// This is the same aura constructor a warrior's own Sunder Armor spell
	// calls when its own spell object is built (sim/warrior can't be
	// imported from sim/core, so this reaches for the core-package aura
	// directly, the way the other tests in this file do).
	sunder := SunderArmorAura(target)

	env.finalize(raidProto, encounterProto, raidStats, false)
	sim := &Simulation{Environment: env}

	beforeArmor := target.stats[stats.Armor]

	// This mirrors the player's rotation landing the hit during combat.
	sunder.Activate(sim)
	sunder.AddStack(sim)

	// HasAura alone would prove nothing here: the aura object was already
	// registered pre-finalize, before Activate ran. HasActiveAura reflects
	// whether the cast actually landed.
	if !target.HasActiveAura("Sunder Armor") {
		t.Fatal("a player-cast Sunder Armor did not land on a dummy target, want it active")
	}
	if got := target.stats[stats.Armor]; got >= beforeArmor {
		t.Errorf("Sunder Armor stack did not lower the dummy's stats.Armor (before=%v, after=%v); a player's own debuff must still take effect even though the raid panel is skipped",
			beforeArmor, got)
	}

	// Switch 3 still holds: even though the player's own debuff just
	// lowered stats.Armor, Armor() (what the damage formula reads) stays
	// pinned to the initial value.
	if got := target.Armor(); got != beforeArmor {
		t.Errorf("Armor() on a dummy = %v after a player-cast Sunder Armor, want the unreduced %v", got, beforeArmor)
	}
}

func dummyEnv(t *testing.T, dummy bool, debuffs *proto.Debuffs) *Environment {
	t.Helper()
	env, _, _ := NewEnvironment(
		SinglePlayerRaidProto(&proto.Player{
			Name:      "Dummy Test",
			Race:      proto.Race_RaceOrc,
			Class:     proto.Class_ClassShaman,
			Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment: &proto.EquipmentSpec{},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, debuffs),
		&proto.Encounter{
			Duration:    180,
			Targets:     []*proto.Target{DefaultTargetProtoLvl60},
			TargetDummy: dummy,
		},
		false,
	)
	return env
}
