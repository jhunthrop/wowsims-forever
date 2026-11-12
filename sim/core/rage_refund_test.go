package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// A rage ability that misses, is dodged or parried gives back 80% of the
// rage it cost (the Era server's rule, which the energy bar already follows:
// energy.go refunds Refund x the cost paid). The refund is of what the cast
// actually spent, after the talents that cut the cost: Improved Heroic
// Strike 3 (12282, -3) leaves Heroic Strike at 12, so a dodged one gives
// back 9.6, not 80% of the base 15; Improved Execute 2 (20502, -5) leaves
// Execute at 10, so a missed one gives back 8, never more than was spent
// (sim-accuracy 2026-11-12, melee-hunters B1).
func TestRageRefundIsEightyPercentOfTheCostPaid(t *testing.T) {
	cases := []struct {
		name string
		base float64
		flat int32
		want float64
	}{
		{"no cut", 15, 0, 12},
		{"Improved Heroic Strike 3", 15, -3, 9.6},
		{"Improved Heroic Strike 1", 15, -1, 11.2},
		{"Improved Execute 2", 15, -5, 8},
	}
	for _, c := range cases {
		warrior := newGnomeEurekaTestCaster(proto.Class_ClassWarrior)
		warrior.Race = proto.Race_RaceHuman
		warrior.EnableRageBar(RageBarOptions{DamageDealtMultiplier: 1, DamageTakenMultiplier: 1})
		spell := &Spell{Unit: &warrior.Unit, ActionID: ActionID{SpellID: 11567}}
		spell.Cost = newRageCost(spell, RageCostOptions{Cost: c.base, Refund: 0.8, RefundMetrics: &ResourceMetrics{}})
		spell.Cost.FlatModifier = c.flat
		spell.CurCast.Cost = spell.Cost.GetCurrentCost()
		if got := spell.Cost.SpellCostFunctions.(*RageCost).RefundAmount(spell); got < c.want-1e-9 || got > c.want+1e-9 {
			t.Errorf("%s: a failed cast at %v rage refunds %v, want %v", c.name, spell.CurCast.Cost, got, c.want)
		}
	}
}
