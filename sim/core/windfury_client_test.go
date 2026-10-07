package core

import (
	"testing"
	"time"
)

// The Windfury Totem buff is read from the 1.60.1.70009 client
// (spellconst/shaman.json, spells 8516/10608/10610): attack power 95, 179
// and 246 for 1 second, one extra attack. Vanilla Classic's 122/229/315
// for 1.5 seconds is not this build's.
func TestWindfuryTotemBuffMatchesClient(t *testing.T) {
	want := [WindfuryRanks + 1]float64{0, 95, 179, 246}
	if WindfuryBuffBonusAP != want {
		t.Errorf("Windfury Totem attack power by rank = %v, want %v", WindfuryBuffBonusAP, want)
	}
	if WindfuryBuffDuration != time.Second {
		t.Errorf("Windfury Totem buff duration = %v, want 1s", WindfuryBuffDuration)
	}
}
