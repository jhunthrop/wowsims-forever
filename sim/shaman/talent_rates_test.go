package shaman

import (
	"math"
	"testing"
	"time"
)

// Forever's 1.60.1.70291 talent text (talents/shaman.json) at max rank:
// Ancestral Knowledge +10% Intellect, Anticipation +6% dodge, Toughness +10%
// Stamina, Flurry +25% attack speed, Improved Fire Nova +20% damage and -4
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
	near("Flurry 1/5", flurryAttackSpeed(1), 1.05)
	near("Flurry 5/5", flurryAttackSpeed(5), 1.25)
	near("Improved Fire Nova 2/2 damage", improvedFireNovaDamagePerPoint*2, 0.20)
	if got := improvedFireNovaCooldownPerPoint * 2; got != 4*time.Second {
		t.Errorf("Improved Fire Nova 2/2 cooldown cut = %v, want 4s", got)
	}
	for rank, want := range []float64{1, 1.07, 1.13, 1.20} {
		near("Elemental Weapons Rockbiter", elementalWeaponsRockbiterBonus[rank], want)
	}
}
