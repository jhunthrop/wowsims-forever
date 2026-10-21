package core

import (
	"testing"
	"time"
)

// A weapon with no swing speed must never be scheduled: its swing
// duration would be zero and the simulation would swing it at the same
// instant forever (the site's rotation ladder hung on exactly this when
// an off-hand held item was equipped as a weapon).
func TestZeroSpeedWeaponNeverSwings(t *testing.T) {
	wa := &WeaponAttack{Weapon: Weapon{SwingSpeed: 0}}
	sim := &Simulation{}
	sim.CurrentTime = 5 * time.Second
	if got := wa.trySwing(sim); got != NeverExpires {
		t.Fatalf("zero-speed weapon scheduled a swing at %v; want NeverExpires", got)
	}
}
