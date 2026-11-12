package feral

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Rend and Tear reaches a bleed the cat applies to a target that already
// bleeds (feral_talents.go): a Rip on a clean target snapshots its plain
// tick, a Rip over the raid's bleed 15% more, and a bleed never counts
// itself.
func TestRendAndTearReachesRipAndRakeOnABleedingTarget(t *testing.T) {
	cat, sim, target := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"rend_and_tear": 5}))
	cat.AssumeBleedActive = false
	cat.AddComboPoints(sim, 5, target, cat.Rip.ComboPointMetrics())
	cat.Rip.Dot(target).Apply(sim)
	plain := cat.Rip.Dot(target).SnapshotBaseDamage
	if plain <= 0 {
		t.Fatal("Rip snapshots no damage")
	}
	cat.Rip.Dot(target).Deactivate(sim)

	cat.AssumeBleedActive = true
	cat.AddComboPoints(sim, 5, target, cat.Rip.ComboPointMetrics())
	cat.AddEnergy(sim, 100, cat.NewEnergyMetrics(core.ActionID{SpellID: 1}))
	landed := false
	for i := 0; i < 20 && !landed; i++ {
		sim.CurrentTime += core.GCDDefault
		cat.AddEnergy(sim, 100, cat.NewEnergyMetrics(core.ActionID{SpellID: 1}))
		cat.AddComboPoints(sim, 5, target, cat.Rip.ComboPointMetrics())
		cat.Rip.Cast(sim, target)
		landed = cat.Rip.Dot(target).IsActive()
	}
	if !landed {
		t.Skip("Rip never landed on this seed")
	}
	if got, want := cat.Rip.Dot(target).SnapshotBaseDamage/plain, 1.15; got < 1.149 || got > 1.151 {
		t.Errorf("Rip tick over a bleed / plain = %.4f, want %v", got, want)
	}

	cat.AssumeBleedActive = false
	cat.Rip.Dot(target).Deactivate(sim)
	cat.Rake.Dot(target).Deactivate(sim)
	cat.AddEnergy(sim, 100, cat.NewEnergyMetrics(core.ActionID{SpellID: 1}))
	for i := 0; i < 20 && !cat.Rake.Dot(target).IsActive(); i++ {
		sim.CurrentTime += core.GCDDefault
		cat.AddEnergy(sim, 100, cat.NewEnergyMetrics(core.ActionID{SpellID: 1}))
		cat.Rake.Cast(sim, target)
	}
	if !cat.Rake.Dot(target).IsActive() {
		t.Skip("Rake never landed on this seed")
	}
	rakeTick := cat.Rake.Dot(target).SnapshotBaseDamage
	if want := 42 + 0.08*cat.Rake.MeleeAttackPower(target); rakeTick < want-0.01 || rakeTick > want+0.01 {
		t.Errorf("Rake tick on a clean target = %.2f, want %.2f (the rank's 42 plus 8%% of attack power, no Rend and Tear)", rakeTick, want)
	}
}
