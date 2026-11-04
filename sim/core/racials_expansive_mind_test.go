package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Expansive Mind is one client row per resource, each aura 178 (maximum
// power percent) at 5: 20591 Mana (misc 0, priest, mage and warlock),
// 1259802 Rage (misc 1, warrior) and 1259803 Energy (misc 3, rogue) in
// 1.60.1.70753. A Gnome warrior's rage bar holds 105 and a Gnome rogue's
// energy bar 105; the racial used to add a Rage and an Energy stat nothing
// reads, so both stayed at 100.
func TestExpansiveMindRaisesMaximumRageAndEnergy(t *testing.T) {
	expansiveMind := racialNamed(t, proto.Race_RaceGnome, "Expansive Mind")

	warrior := newGnomeEurekaTestCaster(proto.Class_ClassWarrior)
	warrior.EnableRageBar(RageBarOptions{DamageDealtMultiplier: 1, DamageTakenMultiplier: 1})
	expansiveMind.Apply(warrior)
	if got := warrior.MaxRage(); got != 105 {
		t.Errorf("a Gnome warrior's maximum rage is %v, want 105 (client 1259802)", got)
	}

	rogue := newGnomeEurekaTestCaster(proto.Class_ClassRogue)
	rogue.EnableEnergyBar(100)
	expansiveMind.Apply(rogue)
	if got := rogue.MaxEnergy(); got != 105 {
		t.Errorf("a Gnome rogue's maximum energy is %v, want 105 (client 1259803)", got)
	}
}

// A rage bar without the racial holds the plain 100.
func TestTheRageBarHoldsOneHundredByDefault(t *testing.T) {
	warrior := newGnomeEurekaTestCaster(proto.Class_ClassWarrior)
	warrior.EnableRageBar(RageBarOptions{DamageDealtMultiplier: 1, DamageTakenMultiplier: 1})
	if got := warrior.MaxRage(); got != MaxRage {
		t.Errorf("the default maximum rage is %v, want %v", got, MaxRage)
	}
}
