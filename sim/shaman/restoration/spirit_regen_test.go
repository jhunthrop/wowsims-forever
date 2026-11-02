package restoration

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// A shaman's spirit regen is 17 + Spirit/5 mana every 2 s tick, two more
// a tick than the druid's and paladin's 15 + Spirit/5. cmangos-classic
// Unit::OCTRegenMPPerSpirit and vmangos Unit::OCTRegenMPPerSpirit both
// carry `case CLASS_SHAMAN: addvalue = (Spirit / 5 + 17)`.
func TestShamanSpiritRegenIsSeventeenPlusAFifthOfSpiritATick(t *testing.T) {
	_, healer := newHealer(t, 60, "")
	spirit := healer.GetStat(stats.Spirit)
	want := (17 + spirit/5) / 2
	got := healer.SpiritManaRegenPerSecondDefault()
	if healer.SpiritManaRegenPerSecond != nil {
		got = healer.SpiritManaRegenPerSecond()
	}
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("spirit regen at %.0f spirit = %.3f mana a second, want (17 + spirit/5)/2 = %.3f", spirit, got, want)
	}
}
