package mage

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// One Missile Barrage proc gives one free Arcane Missiles ("your next Arcane
// Missiles spell"). The consume used to skip any Arcane Missiles completed
// in the same instant the buff was gained, guarding a cast that granted it;
// but Arcane Missiles never grants Missile Barrage, and a channel started at
// the proc's timestamp completes its cast in that same instant, so the first
// free channel kept the buff and the next one was free too (audit X2).
func TestMissileBarrageIsSpentByTheFirstArcaneMissiles(t *testing.T) {
	// The reference Frost build carries Missile Barrage 1/1.
	built, sim, target := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, ForeverFrostTalents)
	if built.MissileBarrageAura == nil {
		t.Fatal("Missile Barrage is not registered with the talent taken")
	}
	am := built.ArcaneMissiles[ArcaneMissilesRanks]

	built.MissileBarrageAura.Activate(sim)
	if !am.Cast(sim, target) {
		t.Fatal("Arcane Missiles did not cast")
	}
	if built.MissileBarrageAura.IsActive() {
		t.Error("Missile Barrage is still active after the Arcane Missiles it made free; one proc must give one free channel")
	}
}
