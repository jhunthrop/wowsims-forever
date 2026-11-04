package tank

import (
	"math"
	"testing"
)

// Genesis (1223081) effect 0 is aura 108 with class mask
// {0x220252, 0x4000100, 0x200000, 0x40000000} (1.60.1.70755 SpellEffect
// 1210794). Lacerate's SpellClassOptions mask is {0, 0x100, 0, 0} on every
// rank (1328617 through 1235827), so its bleed is in the talent's periodic
// damage, as Rake's and Rip's are.
func TestGenesisRaisesLaceratesBleed(t *testing.T) {
	base, _, _ := newBearSim(t, 60, map[string]int{"lacerate": 1})
	talented, _, _ := newBearSim(t, 60, map[string]int{"lacerate": 1, "genesis": 5})

	got := talented.Lacerate.PeriodicDamageMultiplierAdditive
	want := base.Lacerate.PeriodicDamageMultiplierAdditive + 0.10
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("5/5 Genesis Lacerate.PeriodicDamageMultiplierAdditive = %v, want %v", got, want)
	}
}
