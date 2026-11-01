package mage

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// The published level-60 Arcane build (data/builds/1.60.1.70707/bis/
// mage-arcane.json): Arcane Blast and Arcane Power both taken.
const arcanePowerCostTestTalents = "131205213100301531-30000000000000000-053500030000000000"

// Arcane Power (client 12042, effect 1) raises the mana cost of mage
// spells by 30% (misc 14), and the Arcane Blast buff (400573) adds 100% a
// stack. Both are percentage cost mods, so they add. Arcane Blast's stack
// change used to set Cost.Multiplier outright, which dropped Arcane
// Power's +30 for the rest of the window (and took 30 off when Arcane
// Power then ended).
func TestArcanePowerCostSurvivesAnArcaneBlastStack(t *testing.T) {
	built, sim, target := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, arcanePowerCostTestTalents)
	if built.ArcanePowerAura == nil || built.ArcaneBlastAura == nil {
		t.Fatal("the Arcane build did not register Arcane Power and Arcane Blast")
	}
	blast := topRank(t, built.ArcaneBlast)

	built.ArcanePowerAura.Activate(sim)
	blast.ApplyEffects(sim, target, blast)
	if got, want := blast.Cost.Multiplier, int32(100+30+arcaneBlastCostIncreasePctPerStack); got != want {
		t.Errorf("Arcane Blast Cost.Multiplier with Arcane Power and one stack = %d, want %d", got, want)
	}

	built.ArcanePowerAura.Deactivate(sim)
	built.ArcaneBlastAura.Deactivate(sim)
	if got, want := blast.Cost.Multiplier, int32(100); got != want {
		t.Errorf("Arcane Blast Cost.Multiplier after both ended = %d, want %d", got, want)
	}
}
