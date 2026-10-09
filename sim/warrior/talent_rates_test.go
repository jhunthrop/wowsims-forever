package warrior

import (
	"math"
	"testing"
	"time"
)

// Forever's Enrage (1.60.1.70291 talents/warrior.json): "Gives you a 30%
// chance to deal 2/4/6/8/10% increased Physical damage for 12 sec after
// being the victim of any damaging attack."
func TestEnrageRatesAreForevers(t *testing.T) {
	if math.Abs(enrageProcChance-0.30) > 1e-9 {
		t.Errorf("Enrage chance = %v, want 0.30", enrageProcChance)
	}
	if got := enrageDamagePerRank * 5; math.Abs(got-0.10) > 1e-9 {
		t.Errorf("Enrage 5/5 = %v damage, want 0.10", got)
	}
	if enrageDuration != 12*time.Second {
		t.Errorf("Enrage lasts %v, want 12s", enrageDuration)
	}
}
