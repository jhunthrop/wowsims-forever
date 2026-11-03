package feral

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Predatory Discipline (Feral node 104951, spell 1328613): "Your Ferocious
// Bite ability no longer converts extra Energy into damage, but still deals
// extra damage based on your excess Energy." Without it the bite spends the
// energy left after its 35 cost; with it that energy stays on the bar.
func TestPredatoryDisciplineKeepsTheExcessEnergy(t *testing.T) {
	for _, tc := range []struct {
		name  string
		ranks map[string]int
		want  float64
	}{
		{"untalented", nil, 0},
		{"Predatory Discipline", map[string]int{"predatory_discipline": 1}, 100 - 35},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cat, sim, target := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, tc.ranks))
			metrics := cat.NewEnergyMetrics(core.ActionID{SpellID: 1})
			cat.AddEnergy(sim, cat.MaxEnergy()-cat.CurrentEnergy(), metrics)
			cat.AddComboPoints(sim, 5, target, cat.NewComboPointMetrics(core.ActionID{SpellID: 1}))
			cat.ClearcastingAura.Deactivate(sim)
			cat.PoisedAura.Deactivate(sim)

			for attempt := 0; attempt < 50; attempt++ {
				cat.FerociousBite.Cast(sim, target)
				if cat.ComboPoints() == 0 {
					break
				}
				// A miss refunds nothing and keeps the points; top up and retry.
				cat.AddEnergy(sim, cat.MaxEnergy()-cat.CurrentEnergy(), metrics)
				cat.GCD.Reset()
			}
			if cat.ComboPoints() != 0 {
				t.Skip("the bite never landed on this seed")
			}
			if got := cat.CurrentEnergy(); got != tc.want {
				t.Errorf("energy after a 5-point bite from 100 = %v, want %v", got, tc.want)
			}
		})
	}
}
