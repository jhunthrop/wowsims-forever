package druid

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
)

// Bear Form's threat is Bear Form (Passive2) 21178's aura 10 amount: 115% since
// Blizzard's 3 November notes and the 70790 server push 113326.
func TestBearFormThreatMatchesTheClient(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "druid")
	spell, ok := client.ByID(21178)
	if !ok {
		t.Fatal("Bear Form (Passive2) 21178 is not in the client table")
	}
	for _, e := range spell.Effects {
		if e.Aura == 10 {
			if want := 1 + e.Amount/100; math.Abs(bearFormThreatMultiplier-want) > 1e-9 {
				t.Errorf("Bear Form threat %v, client %v", bearFormThreatMultiplier, want)
			}
			if math.Abs(bearFormThreatMultiplier-2.15) > 1e-9 {
				t.Errorf("Bear Form threat %v, Blizzard's 3 Nov notes say +115%%", bearFormThreatMultiplier)
			}
			return
		}
	}
	t.Fatal("21178 has no aura 10 effect")
}
