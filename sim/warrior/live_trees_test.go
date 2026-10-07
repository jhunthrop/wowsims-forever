package warrior

import "testing"

// Dual Wield Specialization's off-hand rage clause is 10% a point
// (client text for build 1.60.1.70009; Blizzard's 1 October 2026 notes:
// "off-hand rage 10/20/30/40/50%").
func TestDualWieldSpecializationOffHandRageMultiplier(t *testing.T) {
	for points, want := range map[int32]float64{1: 1.1, 2: 1.2, 3: 1.3, 4: 1.4, 5: 1.5} {
		if got := dualWieldSpecializationOffHandRageMultiplier(points); got < want-1e-9 || got > want+1e-9 {
			t.Errorf("%d point(s): multiplier = %v, want %v", points, got, want)
		}
	}
}

// Bloodthirst's attack-power ratio: the client text and spell data read
// 35%, Blizzard's 1 October 2026 notes say 45% ("Bloodthirst's
// attack-power ratio 45% (was 35%)"). The note is the live hotfix.
func TestBloodthirstAttackPowerRatioFollowsTheHotfix(t *testing.T) {
	if bloodthirstAttackPowerCoefficient != 0.45 {
		t.Errorf("bloodthirstAttackPowerCoefficient = %v, want 0.45", bloodthirstAttackPowerCoefficient)
	}
}

// Furious Precision's per-rank off-hand hit, from the rank text.
func TestFuriousPrecisionOffHandHitPercent(t *testing.T) {
	for points, want := range map[int32]float64{0: 0, 1: 4, 2: 7, 3: 10, 9: 10} {
		if got := furiousPrecisionOffHandHitPercent(points); got != want {
			t.Errorf("%d point(s): %v%%, want %v%%", points, got, want)
		}
	}
}
