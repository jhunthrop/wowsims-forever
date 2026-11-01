package holy

import (
	"testing"

	"github.com/wowsims/classic/sim/healsim"
)

// Illumination's mana spell 20272 gains, in Forever, effect 1: aura 134
// (mana regeneration while casting) at 50, for 15 s (SpellMisc
// DurationIndex 8; SpellEffect 1237075, 1.60.1.70707). Wowhead Forever's
// buff: "Regenerating at 50% of normal rate while casting spells. 15 seconds
// remaining". Era's 20272 has only the energize. So each Illumination
// proc also opens a 15 s buff.
func TestIlluminationOpensFifteenSecondsOfCastingRegen(t *testing.T) {
	f := numericFight(rotationOf(castItem(divineFavorSpell, tankIndex), castItem(holyLightTopRank, tankIndex)))
	f.talents = map[string]int{"divine_favor": 1, "illumination": maxIlluminationRanks}
	f.duration = 3
	result := f.run(t, numericRuns)

	var procs float64
	for _, aura := range result.RaidMetrics.Parties[0].Players[healsim.HealerIndex].Auras {
		if aura.Id.GetSpellId() == 20272 {
			procs += aura.ProcsAvg
		}
	}
	if procs < 1 {
		t.Errorf("Illumination's casting-regen buff (20272) opened %v times a run on a forced crit, want at least 1", procs)
	}
}
