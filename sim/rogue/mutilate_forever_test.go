package rogue

import "testing"

// Mutilate's flat bonus sits inside its 75% weapon damage, as Backstab's
// sits inside its 150%: client 1241586/1241590 carry effect 121 (weapon
// damage plus 67) with effect 31 (75%), and the tooltip's "plus an
// additional 50 with each weapon" is 67 x 0.75.
func TestMutilateFlatBonusScalesWithItsWeaponPercent(t *testing.T) {
	if got := mutilateHandDamage(67, 100); got != (67+100)*0.75 {
		t.Errorf("Mutilate hand damage with 67 flat on 100 weapon = %v, want (67+100) x 0.75 = %v", got, (67+100)*0.75)
	}
}
