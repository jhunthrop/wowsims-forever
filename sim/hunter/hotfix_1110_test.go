package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Blizzard's 10 November hotfix (us.forums.blizzard.com t/2403040, post 6):
// "Arcane Shot mana cost changed to 11% of base Mana for all ranks of
// Arcane Shot" and "Summon Hawk Mana cost changed to 11% of base Mana for
// all ranks of Summon Hawk, same as Arcane Shot." Client 1.60.1.70888's
// SpellPower rows carry it (PowerCostPct 11, ManaCost 0).
func TestArcaneShotAndSummonHawkCostElevenPercentOfBaseMana(t *testing.T) {
	for _, level := range []int32{20, 40, 60} {
		built := buildHunterWithTalents(t, level, hawkTestTalents(t), proto.Hunter_Options_Cat)
		want := 0.11 * built.BaseMana
		if built.ArcaneShot == nil {
			t.Fatalf("level-%d hunter has no Arcane Shot", level)
		}
		if got := built.ArcaneShot.Cost.BaseCost; math.Abs(got-want) > 1e-6 {
			t.Errorf("level-%d Arcane Shot costs %.2f, want %.2f (11%% of %.0f base mana)", level, got, want, built.BaseMana)
		}
		if built.SummonHawk == nil {
			continue
		}
		if got := built.SummonHawk.Cost.BaseCost; math.Abs(got-want) > 1e-6 {
			t.Errorf("level-%d Summon Hawk costs %.2f, want %.2f (11%% of %.0f base mana)", level, got, want, built.BaseMana)
		}
	}
}
