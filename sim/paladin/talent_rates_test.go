package paladin

import (
	"math"
	"testing"
	"time"
)

// Forever's Vengeance: 1/2/3% Physical and Holy damage a stack for 30 sec
// (20049's curve, 20050 duration index 9). The 70755 hotfix push raised
// 20050's SpellAuraOptions.CumulativeAura from 3 to 5.
func TestVengeanceRatesAreForevers(t *testing.T) {
	if vengeanceDuration != 30*time.Second || vengeanceMaxStacks != 5 {
		t.Errorf("Vengeance lasts %v and stacks to %d, want 30s and 5", vengeanceDuration, vengeanceMaxStacks)
	}
	for rank := int32(1); rank <= 3; rank++ {
		if got, want := vengeanceMultiplier(rank, 1), 1+0.01*float64(rank); math.Abs(got-want) > 1e-9 {
			t.Errorf("Vengeance %d/3 one stack = %v, want %v", rank, got, want)
		}
	}
	if got := vengeanceMultiplier(3, 5); math.Abs(got-1.15) > 1e-9 {
		t.Errorf("Vengeance 3/3 at five stacks = %v, want 1.15", got)
	}
}
