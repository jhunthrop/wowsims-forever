package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Hammer of Wrath is a thrown hammer: SpellMisc gives every rank (24275,
// 24274, 24239) a missile Speed of 35 yards a second, so its damage lands
// after the flight, as Fireball's (24) and Wrath's (20) do.
func TestHammerOfWrathFliesAtTheClientsSpeed(t *testing.T) {
	_, unit := newTalentTestUnit(t, "")
	spell := unit.GetSpell(core.ActionID{SpellID: hammerOfWrathRank3SpellID})
	if spell == nil {
		t.Fatal("Hammer of Wrath rank 3 not registered")
	}
	if got, want := spell.MissileSpeed, 35.0; got != want {
		t.Errorf("Hammer of Wrath missile speed = %v, want the client's %v", got, want)
	}
}
