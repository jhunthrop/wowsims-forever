package feral

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Shifting Power (Feral node 104951, hotfix_only, spell 1322605): "Instantly
// convert [55% of base] Mana into 40 Energy", on a 16 s cooldown, with
// Improved Shifting Power (node 113563, spell 1322670) taking 4/8 s off
// that cooldown. The client's own text prints "0 Mana" because the hotfix
// cost is not in its tables; the 55% and the 16 s are the lane brief's
// figures for the 1 October 2026 hotfix.
func TestShiftingPowerConvertsManaIntoEnergy(t *testing.T) {
	built, sim, target := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"shifting_power": 1}))
	if built.ShiftingPower == nil {
		t.Fatal("a druid with Shifting Power has no Shifting Power spell")
	}

	built.SpendEnergy(sim, built.CurrentEnergy(), built.NewEnergyMetrics(core.ActionID{SpellID: 1}))
	energyBefore, manaBefore := built.CurrentEnergy(), built.CurrentMana()

	built.ShiftingPower.Cast(sim, target)

	if got, want := built.CurrentEnergy()-energyBefore, 40.0; got != want {
		t.Errorf("Shifting Power granted %v energy, want %v", got, want)
	}
	if got, want := manaBefore-built.CurrentMana(), 0.55*built.BaseMana; got < want-0.01 || got > want+0.01 {
		t.Errorf("Shifting Power spent %v mana, want 55%% of base mana (%v)", got, want)
	}
}

func TestShiftingPowerCooldownShrinksWithImprovedShiftingPower(t *testing.T) {
	for points, want := range map[int]time.Duration{0: 16 * time.Second, 1: 12 * time.Second, 2: 8 * time.Second} {
		built, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{
			"shifting_power":          1,
			"improved_shifting_power": points,
		}))
		if got := built.ShiftingPower.CD.Duration; got != want {
			t.Errorf("%d point(s) in Improved Shifting Power: cooldown %s, want %s", points, got, want)
		}
	}
}

func TestShiftingPowerNeedsTheTalent(t *testing.T) {
	built, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	if built.ShiftingPower != nil {
		t.Error("a druid without the talent has a Shifting Power spell")
	}
}
