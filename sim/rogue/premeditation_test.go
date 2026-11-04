package rogue_test

import (
	"testing"
	"time"
)

// Forever's Premeditation (14183) does not need Stealth. Its SpellShapeshift
// row (39913) has ShapeshiftMask 0, where Garrote, Ambush, Cheap Shot and Sap
// carry the Stealth form bit (536870912), and its Attributes_0 (0x50010)
// lacks ONLY_STEALTHED (0x20000), which Garrote's (0x270010) has. Wowhead's
// Forever tooltip (read 2026-11-03) shows no "Requires Stealth" line, which
// its Classic tooltip and Forever's own Garrote tooltip both show. So the
// talent comes off cooldown in combat and can be used again.
func TestPremeditationDoesNotNeedStealth(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, talentsWithRanks(t, map[string]int{"premeditation": 1}), "combat_backstab_prebis")
	rogue := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]
	if rogue.Premeditation == nil {
		t.Fatal("a level 60 rogue with the talent has no Premeditation")
	}
	if rogue.IsStealthed() {
		t.Fatal("the test rogue starts stealthed")
	}
	if !rogue.Premeditation.CanCast(sim, target) {
		t.Fatal("Premeditation is not castable out of stealth")
	}

	rogue.Premeditation.Cast(sim, target)
	if got, want := rogue.ComboPoints(), int32(2); got != want {
		t.Errorf("combo points after Premeditation = %d, want %d", got, want)
	}
	if got, want := rogue.Premeditation.CD.Duration, 2*time.Minute; got != want {
		t.Errorf("Premeditation cooldown %v, want %v", got, want)
	}
}
