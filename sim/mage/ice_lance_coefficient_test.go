package mage

import (
	"math"
	"testing"
)

// TestIceLanceSpellPowerCoefficientFollowsTheLogs: the client's
// coefficient column is 0 for every Ice Lance rank, and the generator
// filled in 0.4286, the instant-spell convention. Forever beta logs say
// otherwise. Four Frost mages at level 33-35 (foreverlogs reports 2762,
// 2763, 2786), each with spell power read off their own Frostbolt, Cone
// of Cold and Frost Nova hits at the client's coefficients (the three
// agree within about 10%), hit with Ice Lance for 0.10 to 0.12 a point of
// spell power, unfrozen and frozen alike. 0.4286 would put those hits
// 40 to 50 damage higher.
func TestIceLanceSpellPowerCoefficientFollowsTheLogs(t *testing.T) {
	_, built := newIgniteMageSim(t)
	// The top rank, which the low-rank penalty leaves whole at 60.
	lance := built.IceLance[len(built.IceLance)-1]
	if lance == nil {
		t.Fatal("level-60 mage registered no top-rank Ice Lance")
	}
	if math.Abs(lance.BonusCoefficient-iceLanceSpellCoeff) > 1e-9 {
		t.Errorf("top-rank Ice Lance coefficient = %v, want %v", lance.BonusCoefficient, iceLanceSpellCoeff)
	}
	if iceLanceSpellCoeff < 0.10 || iceLanceSpellCoeff > 0.12 {
		t.Errorf("iceLanceSpellCoeff = %v, outside the logs' 0.10 to 0.12", iceLanceSpellCoeff)
	}
}
