package elemental

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// TestShamanSpiritRegenIsSeventeenPlusAFifth: the Classic server gives a
// shaman 17 + Spirit/5 mana a 2 s tick out of the five-second rule (cmangos
// Unit::OCTRegenMPPerSpirit, "case CLASS_SHAMAN: addvalue = (Spirit / 5 +
// 17)"; mangoszero and AscEmu say the same), where druids, hunters,
// paladins and warlocks get 15 + Spirit/5. The shaman read the 15 default.
func TestShamanSpiritRegenIsSeventeenPlusAFifth(t *testing.T) {
	_, s := newFireNovaShaman(t, 60, "")
	spirit := s.GetStat(stats.Spirit)
	got := s.SpiritManaRegenPerSecondDefault()
	if s.SpiritManaRegenPerSecond != nil {
		got = s.SpiritManaRegenPerSecond()
	}
	want := (17 + spirit/5) / 2
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("shaman spirit regen = %.4f mana a second at %.0f Spirit, want %.4f (17 + Spirit/5 a 2 s tick)", got, spirit, want)
	}
}
