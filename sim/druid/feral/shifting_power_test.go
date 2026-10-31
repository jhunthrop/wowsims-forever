package feral

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Shifting Power (Feral node 113563, spell 1322605): "Instantly convert
// [55% of base] Mana into 40 Energy". The client's own text prints "0 Mana"
// because the hotfix cost is not in its tables; the 55% is the tables' cost
// since build 1.60.1.70291. Client 1.60.1.70707 removed Improved Shifting
// Power and the live tooltip states an 8 s cooldown.
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

func TestShiftingPowerNeedsTheTalent(t *testing.T) {
	built, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	if built.ShiftingPower != nil {
		t.Error("a druid without the talent has a Shifting Power spell")
	}
}

func TestShiftingPowerCooldownIsEightSeconds(t *testing.T) {
	built, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"shifting_power": 1}))
	if got, want := built.ShiftingPower.CD.Duration, 8*time.Second; got != want {
		t.Errorf("Shifting Power cooldown = %v, want %v", got, want)
	}
}
