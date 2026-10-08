package rogue_test

import (
	"testing"
	"time"
)

// landOnce drives apply until the rogue holds a combo point, the sign the
// cast landed (misses and dodges refund and add none), so a fixed seed's
// bad roll cannot fail a test of what a landed cast does.
func landOnce(t *testing.T, apply func(), comboPoints func() int32) {
	t.Helper()
	for attempt := 0; attempt < 50; attempt++ {
		apply()
		if comboPoints() > 0 {
			return
		}
	}
	t.Fatal("no cast landed in 50 attempts")
}

func TestGougeDealsClientDamageAndGrantsOneComboPoint(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis")
	rogue := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]
	if rogue.Gouge == nil {
		t.Fatal("a level 60 rogue has no Gouge")
	}
	if got, want := rogue.Gouge.ActionID.SpellID, int32(11286); got != want {
		t.Errorf("Gouge is spell %d, want the top rank %d", got, want)
	}
	if got, want := rogue.Gouge.CD.Duration, 10*time.Second; got != want {
		t.Errorf("Gouge cooldown %v, want %v", got, want)
	}

	landOnce(t, func() { rogue.Gouge.ApplyEffects(sim, target, rogue.Gouge) }, rogue.ComboPoints)
	if got, want := rogue.ComboPoints(), int32(1); got != want {
		t.Errorf("combo points after a landed Gouge = %d, want %d", got, want)
	}
	if rogue.Gouge.SpellMetrics[target.UnitIndex].TotalDamage <= 0 {
		t.Error("Gouge dealt no damage")
	}
}

func TestKickDealsClientDamage(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis")
	rogue := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]
	if rogue.Kick == nil {
		t.Fatal("a level 60 rogue has no Kick")
	}
	if got, want := rogue.Kick.ActionID.SpellID, int32(1769); got != want {
		t.Errorf("Kick is spell %d, want the top rank %d", got, want)
	}

	for attempt := 0; attempt < 50 && rogue.Kick.SpellMetrics[target.UnitIndex].TotalDamage == 0; attempt++ {
		rogue.Kick.ApplyEffects(sim, target, rogue.Kick)
	}
	if rogue.Kick.SpellMetrics[target.UnitIndex].TotalDamage <= 0 {
		t.Error("Kick dealt no damage in 50 attempts")
	}
}

func TestCheapShotNeedsStealthAndGrantsTwoComboPoints(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis")
	rogue := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]
	if rogue.CheapShot == nil {
		t.Fatal("a level 60 rogue has no Cheap Shot")
	}

	if rogue.CheapShot.CanCast(sim, target) {
		t.Fatal("Cheap Shot is castable out of stealth")
	}
	rogue.StealthAura.Activate(sim)
	if !rogue.CheapShot.CanCast(sim, target) {
		t.Fatal("Cheap Shot is not castable from stealth")
	}

	landOnce(t, func() {
		rogue.StealthAura.Activate(sim)
		rogue.CheapShot.ApplyEffects(sim, target, rogue.CheapShot)
	}, rogue.ComboPoints)
	if got, want := rogue.ComboPoints(), int32(2); got != want {
		t.Errorf("combo points after a landed Cheap Shot = %d, want %d", got, want)
	}
	if rogue.StealthAura.IsActive() {
		t.Error("stealth is still active after Cheap Shot")
	}
}
