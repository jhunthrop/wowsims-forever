package hunter

import "testing"

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
