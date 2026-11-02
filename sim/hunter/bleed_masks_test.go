package hunter

import (
	"math"
	"testing"
)

// Round 10 (physical lane): Lacerate (1299332) and Lacerating Strikes'
// bleed (1310536) share one class flag, mask 1 bit 4096. Savage Strikes
// (19159, aura 107 misc 7, crit chance: 66 / 528384 / 2097152) and
// Predator's Edge (1310627, aura 108 misc 15, crit damage: 66 / 528384 /
// 1075904512) both carry that bit, so both bleeds take Savage Strikes'
// crit and Predator's Edge's crit damage. The engine gave Lacerate the
// crit only and Lacerating Strikes neither.
func TestTheBleedsTakeSavageStrikesAndPredatorsEdge(t *testing.T) {
	_, built, _ := buildSurvivalMeleeHunter(t)
	crit := built.savageStrikesCritRating()
	critDamage := built.predatorsEdgeCritDamage()
	if crit <= 0 || critDamage <= 0 {
		t.Fatalf("the test build needs Savage Strikes and Predator's Edge (crit %v, crit damage %v)", crit, critDamage)
	}
	for _, c := range []struct {
		name string
		got  func() (float64, float64, bool)
	}{
		{"Lacerate", func() (float64, float64, bool) {
			if built.Lacerate == nil {
				return 0, 0, false
			}
			return built.Lacerate.BonusCritRating, built.Lacerate.CritDamageBonus, true
		}},
		{"Lacerating Strikes", func() (float64, float64, bool) {
			if built.LaceratingStrikes == nil {
				return 0, 0, false
			}
			return built.LaceratingStrikes.BonusCritRating, built.LaceratingStrikes.CritDamageBonus, true
		}},
	} {
		gotCrit, gotDamage, ok := c.got()
		if !ok {
			t.Errorf("%s is not registered on the test build", c.name)
			continue
		}
		if math.Abs(gotCrit-crit) > 1e-9 {
			t.Errorf("%s bonus crit rating = %v, want Savage Strikes' %v", c.name, gotCrit, crit)
		}
		if math.Abs(gotDamage-(1+critDamage)) > 1e-9 {
			t.Errorf("%s crit damage bonus = %v, want 1 + Predator's Edge's %v", c.name, gotDamage, critDamage)
		}
	}
}
