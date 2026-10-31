package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// wandSpecializationMultiplier is the class-specific half of core.RegisterShootSpell's wand
// damage math; core/wand_test.go covers the shared Shoot spell config directly (school, zero
// cost, cast time from a known wand's speed), so this only needs the talent-rank table.
func TestWandSpecializationMultiplier(t *testing.T) {
	cases := []struct {
		rank int32
		want float64
	}{
		{rank: 0, want: 1},
		{rank: 1, want: 1.13},
		{rank: 2, want: 1.25},
	}

	for _, c := range cases {
		mage := &Mage{Talents: &proto.MageTalents{WandSpecialization: c.rank}}
		if got := mage.wandSpecializationMultiplier(); got != c.want {
			t.Errorf("wandSpecializationMultiplier() at rank %d = %v, want %v", c.rank, got, c.want)
		}
	}
}

// TestShootDealsWandDamageAgainstARealTarget is the end-to-end half: newFrostMageSim (see
// frost_nova_test.go) builds a real mage through the normal agent/Initialize() path, which
// confirms Shoot is correctly absent with the empty gear this test environment's fork of the
// item database forces (frost_nova_test.go/pet_level_test.go). Injecting an Item directly and
// re-running the exact registerShootSpell() Initialize() already called once exercises
// core.RegisterShootSpell's real damage path - spell hit/crit tables, no mana cost - against
// a live target, which the pure-math checks in core/wand_test.go can't reach on their own.
func TestShootDealsWandDamageAgainstARealTarget(t *testing.T) {
	built, sim, target := newFrostMageSim(t, core.DefaultTargetProtoLvl60)

	if built.Shoot != nil {
		t.Fatal("Shoot was registered even though this test's EquipmentSpec leaves the ranged slot empty")
	}

	built.Equipment[proto.ItemSlot_ItemSlotRanged] = core.Item{
		ID:               99999,
		RangedWeaponType: proto.RangedWeaponType_RangedWeaponTypeWand,
		WeaponDamageMin:  20,
		WeaponDamageMax:  40,
		SwingSpeed:       1.5,
	}
	built.registerShootSpell()
	if built.Shoot == nil {
		t.Fatal("registerShootSpell did not register Shoot once a wand was equipped")
	}

	const iterations = 200
	for i := 0; i < iterations; i++ {
		built.Shoot.ApplyEffects(sim, target, built.Shoot)
	}
	waitForOutcomes(sim)

	metrics := built.Shoot.SpellMetrics[target.UnitIndex]
	if metrics.TotalDamage <= 0 {
		t.Fatalf("Shoot dealt no damage over %d casts against a level-60 target", iterations)
	}
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatal("Shoot never landed a hit or a crit")
	}
}

// A wand shot is a cast of its own: it never starts inside another spell's
// global cooldown, and once started it holds the caster for the wand's whole
// speed. Before this held, the rotation fired Shoot in the same instant as
// every instant spell (Fire Blast, Corruption, Shadow Word: Pain), dealt a
// free wand hit and let the next spell start before that GCD had run out:
// a 1.2 s wand ended a 1.5 s GCD 0.3 s early.
func TestShootNeverShortensTheGlobalCooldown(t *testing.T) {
	built, sim, target := newFrostMageSim(t, core.DefaultTargetProtoLvl60)
	built.Equipment[proto.ItemSlot_ItemSlotRanged] = core.Item{
		ID:               99999,
		RangedWeaponType: proto.RangedWeaponType_RangedWeaponTypeWand,
		WeaponDamageMin:  20,
		WeaponDamageMax:  40,
		SwingSpeed:       1.2,
	}
	built.registerShootSpell()

	if !built.FireBlast[len(built.FireBlast)-1].Cast(sim, target) {
		t.Fatal("Fire Blast did not cast")
	}
	gcdEnd := built.GCD.ReadyAt()
	if gcdEnd <= sim.CurrentTime {
		t.Fatalf("Fire Blast left no GCD running (ready at %v, now %v)", gcdEnd, sim.CurrentTime)
	}
	if built.Shoot.CanCast(sim, target) {
		t.Fatalf("Shoot is castable %v into Fire Blast's GCD; a wand shot must wait for the GCD", sim.CurrentTime)
	}

	sim.CurrentTime = gcdEnd
	if !built.Shoot.Cast(sim, target) {
		t.Fatal("Shoot did not cast once the GCD was ready")
	}
	want := gcdEnd + time.Duration(1.2*float64(time.Second))
	if got := built.Hardcast.Expires; got != want {
		t.Errorf("Shoot's cast ends at %v, want %v (the wand's full speed)", got, want)
	}
	if got := built.GCD.ReadyAt(); got < want {
		t.Errorf("the next spell may start at %v, before the wand shot ends at %v", got, want)
	}
}
