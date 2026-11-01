package druid

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func nearly(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Forever's druid deep dive: a form's auto attack has the equipped
// weapon's DPS at 1.0 s (Cat) or 2.5 s (Bear), and weapon-damage
// abilities use the same values.
func TestFormPawCarriesTheEquippedWeaponsDPS(t *testing.T) {
	staff := core.Weapon{BaseDamageMin: 100, BaseDamageMax: 160, SwingSpeed: 3.2, NormalizedSwingSpeed: 3.3, TwoHanded: true}
	for _, speed := range []float64{catPawSwingSpeed, bearPawSwingSpeed} {
		paw := formPawWeapon(staff, speed)
		if !nearly(paw.DPS(), staff.DPS()) {
			t.Errorf("speed %.1f: paw DPS %.3f, want the weapon's %.3f", speed, paw.DPS(), staff.DPS())
		}
		if paw.SwingSpeed != speed || paw.NormalizedSwingSpeed != speed {
			t.Errorf("paw speeds %.1f/%.1f, want %.1f", paw.SwingSpeed, paw.NormalizedSwingSpeed, speed)
		}
		if !nearly(paw.BaseDamageMin, 100/3.2*speed) || !nearly(paw.BaseDamageMax, 160/3.2*speed) {
			t.Errorf("speed %.1f: paw range %.2f-%.2f keeps the weapon's spread at %.2f-%.2f", speed, paw.BaseDamageMin, paw.BaseDamageMax, 100/3.2*speed, 160/3.2*speed)
		}
		if paw.TwoHanded {
			t.Error("the paw is not a two-hander for rage normalization")
		}
	}
}

func TestFormPawUnarmedKeepsTheEraPaw(t *testing.T) {
	paw := formPawWeapon(core.Weapon{SwingSpeed: 1}, catPawSwingSpeed)
	if !nearly(paw.DPS(), formPawDamagePerSecond) {
		t.Errorf("unarmed paw DPS %.2f, want %.1f", paw.DPS(), formPawDamagePerSecond)
	}
}
