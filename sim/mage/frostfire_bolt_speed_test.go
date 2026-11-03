package mage

import "testing"

// frostfireBoltClientSpeed is SpellMisc.Speed for every learnable Frostfire
// Bolt rank (401502, 1237312, 1237313) in client 1.60.1.70753: 24 yards a
// second, Fireball's and Pyroblast's speed, not Frostbolt's 28.
const frostfireBoltClientSpeed = 24

func TestFrostfireBoltTravelsAtTheClientSpeed(t *testing.T) {
	_, mage := newMageAtLevel(t, 60, "")
	ranks := registeredFrostfireRanks(mage)
	if len(ranks) == 0 {
		t.Fatal("no Frostfire Bolt rank registered at 60")
	}
	for _, rank := range ranks {
		if got := mage.FrostfireBolt[rank].MissileSpeed; got != frostfireBoltClientSpeed {
			t.Errorf("Frostfire Bolt rank %d flies at %v yd/s, want the client's %d", rank, got, frostfireBoltClientSpeed)
		}
	}
}
