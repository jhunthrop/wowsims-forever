package item_effects_test

// Coverage for leveling_procs.go's three new weapon procs (this lane's
// step 4, 2026-09-28 weights-effects): a level-60 character with the
// item equipped has the proc spell registered, and casting it directly
// (bypassing the PPM roll itself, which is core's own, already-tested
// machinery -- see itemhelpers.CreateWeaponProcSpell) deals the damage
// leveling_procs.go's own doc says this build's item data promises.
//
// package item_effects_test (external, not item_effects) so this can
// import github.com/wowsims/classic/sim (RegisterAll) without an import
// cycle: that package itself blank-imports sim/common, which is what
// wires leveling_procs.go's init() into ItemsByID's item database, and
// RegisterAll's own agent factories (sim/hunter, sim/warrior/dps_warrior)
// are what a real Warrior/Hunter equip needs -- the same entry point
// sim/cmd/leveling-bis's own registerEngine() calls on the site side.

import (
	"testing"

	engine "github.com/wowsims/classic/sim"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// buildCharacterWithWeapon equips itemID in slot (main-hand or ranged)
// on a fresh level-60 character of class, and returns the running
// Simulation plus that character -- enough to look a registered item
// spell up by ActionID and force-cast it.
func buildCharacterWithWeapon(t *testing.T, class proto.Class, spec any, slot proto.ItemSlot, itemID int32) (*core.Simulation, *core.Character) {
	t.Helper()
	engine.RegisterAll()

	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[slot] = &proto.ItemSpec{Id: itemID}

	player := &proto.Player{
		Name:      "Item Effect Test",
		Race:      proto.Race_RaceOrc,
		Class:     class,
		Level:     60,
		Equipment: &proto.EquipmentSpec{Items: items},
		Buffs:     core.FullBuffs.Player,
		Rotation:  &proto.APLRotation{},
	}
	switch s := spec.(type) {
	case *proto.Player_Warrior:
		player.Spec = s
	case *proto.Player_Hunter:
		player.Spec = s
	default:
		t.Fatalf("buildCharacterWithWeapon: unhandled spec type %T", spec)
	}

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

	character := sim.Raid.Parties[0].Players[0].GetCharacter()
	return sim, character
}

func TestFrostTigerBladeDealsDamageWhenCast(t *testing.T) {
	sim, character := buildCharacterWithWeapon(t, proto.Class_ClassWarrior, &proto.Player_Warrior{Warrior: &proto.Warrior{Options: &proto.Warrior_Options{}}}, proto.ItemSlot_ItemSlotMainHand, 3854)

	spell := character.GetSpell(core.ActionID{ItemID: 3854})
	if spell == nil {
		t.Fatal("Frost Tiger Blade equipped, but no proc spell registered for its item ID -- leveling_procs.go's init() did not run or the ActionID does not match")
	}

	target := sim.Encounter.TargetUnits[0]
	spell.Cast(sim, target)

	metrics := spell.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Frost Tiger Blade's proc landed neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Frost Tiger Blade's proc dealt %v damage, want > 0 (item data: 50 Frost damage)", metrics.TotalDamage)
	}
}

func TestBlightDealsInstantAndPeriodicDamageWhenCast(t *testing.T) {
	sim, character := buildCharacterWithWeapon(t, proto.Class_ClassWarrior, &proto.Player_Warrior{Warrior: &proto.Warrior{Options: &proto.Warrior_Options{}}}, proto.ItemSlot_ItemSlotMainHand, 7959)

	spell := character.GetSpell(core.ActionID{SpellID: 9796})
	if spell == nil {
		t.Fatal("Blight equipped, but no proc spell registered for spell id 9796 -- leveling_procs.go's init() did not run or the ActionID does not match")
	}

	target := sim.Encounter.TargetUnits[0]
	spell.Cast(sim, target)

	metrics := spell.SpellMetrics[target.UnitIndex]
	if metrics.TotalDamage <= 0 {
		t.Errorf("Blight's instant hit dealt %v damage, want > 0 (item data: 100 Nature damage)", metrics.TotalDamage)
	}

	dot := spell.Dot(target)
	if dot == nil || !dot.IsActive() {
		t.Fatal("Blight did not apply its disease DoT to the target")
	}
	if dot.SnapshotBaseDamage <= 0 {
		t.Errorf("Blight's DoT snapshot damage = %v, want > 0 (item data: 360 damage over 1 min across 12 ticks)", dot.SnapshotBaseDamage)
	}
}

func TestDarkIronRifleDealsDamageWhenCast(t *testing.T) {
	sim, character := buildCharacterWithWeapon(t, proto.Class_ClassHunter, &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{}}}, proto.ItemSlot_ItemSlotRanged, 16004)

	spell := character.GetSpell(core.ActionID{ItemID: 16004})
	if spell == nil {
		t.Fatal("Dark Iron Rifle equipped, but no proc spell registered for its item ID -- leveling_procs.go's init() did not run or the ActionID does not match")
	}

	target := sim.Encounter.TargetUnits[0]
	spell.Cast(sim, target)

	metrics := spell.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Dark Iron Rifle's proc landed neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Dark Iron Rifle's proc dealt %v damage, want > 0 (item data: 26 Shadow damage)", metrics.TotalDamage)
	}
}
