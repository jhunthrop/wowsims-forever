package holy

import (
	"testing"

	"github.com/wowsims/classic/sim/healsim"
)

// Illumination is mana and nothing else on the live server: its energize
// is 1310908 (Forever's own id, effect 30 alone), and no 20272 aura, the
// client row that carries a 50% casting regen for 15 s (effect 1, aura
// 134), is ever applied. A logged Holy paladin (foreverlogs.gg 2834,
// 24 crit heals, 24 Illumination gains of half the base cost, every one
// from 1310908, no Illumination buff in her buff stream) is the source.
func TestIlluminationGivesManaAndNoCastingRegen(t *testing.T) {
	f := numericFight(rotationOf(castItem(divineFavorSpell, tankIndex), castItem(holyLightTopRank, tankIndex)))
	f.talents = map[string]int{"divine_favor": 1, "illumination": maxIlluminationRanks}
	f.duration = 3
	result := f.run(t, numericRuns)

	player := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
	for _, aura := range player.Auras {
		if aura.Id.GetSpellId() == 20272 && aura.ProcsAvg > 0 {
			t.Errorf("Illumination opened the 20272 casting-regen buff %v times a run; the live proc is mana alone", aura.ProcsAvg)
		}
	}
	var gained float64
	for _, resource := range player.Resources {
		if resource.Id.GetSpellId() == illuminationAction {
			gained += resource.ActualGain
		}
	}
	if gained <= 0 {
		t.Errorf("Illumination returned no mana on a forced crit")
	}
}
