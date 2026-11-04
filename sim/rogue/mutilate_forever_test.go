package rogue

import "testing"

// Mutilate's flat bonus sits inside its 90% weapon damage, as Backstab's
// sits inside its 150%: client 399960/399961 carry effect 121 (weapon
// damage plus 44) with effect 31 (90% in 70755), and the tooltip's "plus
// an additional 40 with each weapon" is 44 x 0.90.
func TestMutilateFlatBonusScalesWithItsWeaponPercent(t *testing.T) {
	if got, want := mutilateHandDamage(44, 100), (44+100)*0.90; got != want {
		t.Errorf("Mutilate hand damage with 44 flat on 100 weapon = %v, want (44+100) x 0.90 = %v", got, want)
	}
	if got, want := MutilateDamage[1].Amount, 44.0; got != want {
		t.Errorf("Mutilate rank 1 flat = %v, want %v", got, want)
	}
}
