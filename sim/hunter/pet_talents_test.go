package hunter

import (
	"testing"
)

// Forever's pet talent rates (1.60.1.70291 talents/hunter.json): Ferocity
// "increases the critical strike chance of your pets and hawks by 10%" at
// 5/5 and Unleashed Fury "increases the damage done by your pets and hawks
// by 15%" at 5/5. Vanilla's 3% and 4% a rank are not Forever's.
func TestPetTalentRatesAreForevers(t *testing.T) {
	if got := ferocityCritPerRank * 5; got != 10 {
		t.Errorf("Ferocity 5/5 = %v%% crit, want 10", got)
	}
	if got := unleashedFuryPerRank * 5; got < 0.1499 || got > 0.1501 {
		t.Errorf("Unleashed Fury 5/5 = %v, want 0.15", got)
	}
}

// Forever's Efficiency, Barrage and Savage Strikes rates (1.60.1.70291
// talents/hunter.json): Efficiency "Reduces the Mana cost of your Shots,
// Stings, and melee abilities by 15%" at 5/5, Barrage "Increases the damage
// done by your Multi-Shot, Aimed Shot, and Volley abilities by 10%" at 3/3,
// Savage Strikes "Increases the critical strike chance of all your melee
// abilities by 4%" at 2/2.
func TestShotTalentRatesAreForevers(t *testing.T) {
	if got := efficiencyCostReductionPerRank * 5; got != 15 {
		t.Errorf("Efficiency 5/5 = %v%% cost reduction, want 15", got)
	}
	for rank, want := range []float64{0, 0.03, 0.07, 0.10} {
		if got := barrageDamage(int32(rank)); got != want {
			t.Errorf("Barrage %d/3 = %v damage, want %v", rank, got, want)
		}
	}
	if got := savageStrikesCritPerRank * 2; got != 4 {
		t.Errorf("Savage Strikes 2/2 = %v%% crit, want 4", got)
	}
}
