package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/rogue"
)

// TestColdBloodOnlyCoversTheTooltipsFiveSpells pins the client's Cold Blood
// text: "your next Sinister Strike, Backstab, Ambush, Eviscerate, or
// Mutilate". Hemorrhage and Ghostly Strike are builders too, but a Cold
// Blood must wait through them (and so must Garrote).
func TestColdBloodOnlyCoversTheTooltipsFiveSpells(t *testing.T) {
	_, built := buildRogueForTest(t, rogueTalentStringWith(t, "mutilate", "hemorrhage", "ghostly_strike"))

	covered := map[string]bool{
		"Sinister Strike": built.SinisterStrike != nil && built.SinisterStrike.Flags.Matches(rogue.SpellFlagColdBlooded),
		"Backstab":        built.Backstab != nil && built.Backstab.Flags.Matches(rogue.SpellFlagColdBlooded),
		"Ambush":          built.Ambush != nil && built.Ambush.Flags.Matches(rogue.SpellFlagColdBlooded),
		"Eviscerate":      built.Eviscerate != nil && built.Eviscerate.Flags.Matches(rogue.SpellFlagColdBlooded),
		"Mutilate":        built.Mutilate != nil && built.Mutilate.Flags.Matches(rogue.SpellFlagColdBlooded),
	}
	for name, ok := range covered {
		if !ok {
			t.Errorf("%s is not Cold Blood-covered", name)
		}
	}
	if built.Hemorrhage == nil || built.GhostlyStrike == nil {
		t.Fatal("test rogue lacks Hemorrhage or Ghostly Strike")
	}
	if built.Hemorrhage.Flags.Matches(rogue.SpellFlagColdBlooded) {
		t.Error("Hemorrhage consumes Cold Blood; the client text does not list it")
	}
	if built.GhostlyStrike.Flags.Matches(rogue.SpellFlagColdBlooded) {
		t.Error("Ghostly Strike consumes Cold Blood; the client text does not list it")
	}
}

// TestSliceAndDiceHastesMeleeSwingsForTheClientDuration pins the client's
// rank 2 text: +30% melee attack speed, 9/12/15/18/21 s for 1-5 combo points.
func TestSliceAndDiceHastesMeleeSwingsForTheClientDuration(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t))
	target := sim.Encounter.TargetUnits[0]
	before := built.PseudoStats.MeleeSpeedMultiplier

	built.AddComboPoints(sim, 3, target, built.SliceAndDice.ComboPointMetrics())
	built.SliceAndDice.ApplyEffects(sim, target, built.SliceAndDice)

	if got, want := built.SliceAndDiceAura.Duration.Seconds(), 15.0; got != want {
		t.Errorf("Slice and Dice at 3 combo points lasts %vs, want %vs", got, want)
	}
	if got, want := built.PseudoStats.MeleeSpeedMultiplier/before, 1.3; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("melee speed multiplier ratio = %v, want %v", got, want)
	}
}
