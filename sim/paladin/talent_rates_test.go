package paladin

import (
	"math"
	"testing"
	"time"
)

// Forever's Vengeance and Vindication (1.60.1.70291 talents/paladin.json):
// Vengeance "Increases your Physical and Holy damage dealt by 1/2/3% for 30
// sec ... Stacks up to 3 times", Vindication "increase your Attack Power by
// 1/2/3% for 30 sec".
func TestVengeanceRatesAreForevers(t *testing.T) {
	if vengeanceDuration != 30*time.Second || vengeanceMaxStacks != 3 {
		t.Errorf("Vengeance lasts %v and stacks to %d, want 30s and 3", vengeanceDuration, vengeanceMaxStacks)
	}
	for rank := int32(1); rank <= 3; rank++ {
		if got, want := vengeanceMultiplier(rank, 1), 1+0.01*float64(rank); math.Abs(got-want) > 1e-9 {
			t.Errorf("Vengeance %d/3 one stack = %v, want %v", rank, got, want)
		}
	}
	if got := vengeanceMultiplier(3, 3); math.Abs(got-1.09) > 1e-9 {
		t.Errorf("Vengeance 3/3 at three stacks = %v, want 1.09", got)
	}
}

func TestVindicationAttackPowerIsForevers(t *testing.T) {
	for rank := int32(0); rank <= 3; rank++ {
		if got, want := vindicationAttackPowerMultiplier(rank), 1+0.01*float64(rank); math.Abs(got-want) > 1e-9 {
			t.Errorf("Vindication %d/3 = %v, want %v", rank, got, want)
		}
	}
}
