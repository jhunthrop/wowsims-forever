package shaman

import (
	"math"
	"testing"
	"time"
)

// Forever's 1.60.1.70291 talent text (talents/shaman.json) at max rank:
// Ancestral Knowledge +10% Intellect, Anticipation +6% dodge, Toughness +10%
// Stamina, Improved Fire Nova +20% damage and -4
// sec cooldown, Elemental Weapons +7/13/20% Rockbiter attack power.
func TestShamanTalentRatesAreForevers(t *testing.T) {
	near := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	near("Ancestral Knowledge 5/5", ancestralKnowledgeIntellectPerRank*5, 0.10)
	near("Anticipation 3/3", anticipationDodgePerRank*3, 6)
	near("Toughness 5/5", toughnessStaminaPerRank*5, 0.10)
	// Flurry is trait curve 85300 (spell 16256 effect 0) in 1.60.1.70755:
	// 6/12/18/24/30% attack speed.
	near("Flurry 1/5", flurryAttackSpeed(1), 1.06)
	near("Flurry 5/5", flurryAttackSpeed(5), 1.30)
	near("Trial by Fire 2/2 damage", trialByFireDamagePerPoint*2, 0.20)
	if got := trialByFireCooldownPerPoint * 2; got != 4*time.Second {
		t.Errorf("Trial by Fire 2/2 cooldown cut = %v, want 4s", got)
	}
	if got := trialByFireMagmaDurationPerPoint * 2; got != 20*time.Second {
		t.Errorf("Trial by Fire 2/2 Magma Totem duration gain = %v, want 20s", got)
	}
	for rank, want := range []float64{1, 1.07, 1.13, 1.20} {
		near("Elemental Weapons Rockbiter", elementalWeaponsRockbiterBonus[rank], want)
	}
}
