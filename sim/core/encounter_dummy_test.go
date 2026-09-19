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
