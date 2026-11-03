package druid

import "testing"

// Enrage's client text (5229, and Wowhead Forever's tooltip, read
// 2026-11-03): "reduces base armor by 27% in Bear Form and 16% in Dire Bear
// Form". A druid of 40 and up tanks in Dire Bear Form (9634).
func TestEnrageArmorPenaltyFollowsTheForm(t *testing.T) {
	for _, tc := range []struct {
		level int32
		want  float64
	}{
		{10, 0.73},
		{39, 0.73},
		{40, 0.84},
		{60, 0.84},
	} {
		tier, ok := bearFormTierAt(tc.level)
		if !ok {
			t.Fatalf("no bear form at %d", tc.level)
		}
		if got := tier.EnrageArmorMultiplier(); got != tc.want {
			t.Errorf("level %d: Enrage leaves %v of the form's base armor, want %v", tc.level, got, tc.want)
		}
	}
}
