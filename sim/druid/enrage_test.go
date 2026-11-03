package druid

import "testing"

// Enrage "reduces base armor by 27% in Bear Form and 16% in Dire Bear Form"
// (Spell 5229 Description_lang, client 1.60.1.70753; the Wowhead Forever
// tooltip reads the same). cmangos-classic casts 25503 (aura 101, base
// armor percent) at -27, or -16 while the caster has Dire Bear Form 9634
// (SpellEffects.cpp, case 5229). Dire Bear Form is learned at 40.
func TestEnrageArmorReductionFollowsTheForm(t *testing.T) {
	for _, c := range []struct {
		level int32
		want  float64
	}{
		{12, 0.27},
		{39, 0.27},
		{40, 0.16},
		{60, 0.16},
	} {
		if got := enrageBaseArmorReductionAt(c.level); got != c.want {
			t.Errorf("level %d: Enrage cuts base armor by %.2f, want %.2f", c.level, got, c.want)
		}
	}
}
