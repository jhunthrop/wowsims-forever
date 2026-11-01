package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's heals over time read the caster's healing power at every tick,
// as its damage over time does: Blizzard's reply on forever-bugs #137
// (2026-10-01) speaks of "DoT/HoT boosting effects that apply dynamically"
// under "dynamic DoTs". A DotConfig marked DynamicSnapshot re-runs its own
// OnSnapshot before each tick, so healing power gained after the cast
// reaches the ticks still to come.
func TestDynamicHotReadsHealingPowerEachTick(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dynamic bool
		gain    float64
	}{
		{"dynamic", true, 100},
		{"snapshot", false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, caster, _ := newPeriodicCritFixture(t, 0)
			target := caster
			spell := caster.RegisterSpell(SpellConfig{
				ActionID:         ActionID{SpellID: 9841},
				SpellSchool:      SpellSchoolNature,
				DefenseType:      DefenseTypeMagic,
				Flags:            SpellFlagHelpful,
				ProcMask:         ProcMaskSpellHealing,
				DamageMultiplier: 1,
				ThreatMultiplier: 1,
				Hot: DotConfig{
					Aura:             Aura{Label: "Test Dynamic Hot"},
					NumberOfTicks:    2,
					TickLength:       time.Second * 3,
					BonusCoefficient: 1,
					DynamicSnapshot:  tc.dynamic,
					OnSnapshot: func(_ *Simulation, target *Unit, dot *Dot, _ bool) {
						dot.SnapshotBaseDamage = 100 + dot.BonusCoefficient*dot.Spell.HealingPower(target)
						dot.SnapshotAttackerMultiplier = 1
					},
					OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
						dot.CalcAndDealPeriodicSnapshotHealing(sim, target, dot.OutcomeTick)
					},
				},
			})
			advanceFixture(sim, spell.Hot(target))
			dot := spell.Hot(target)
			dot.Apply(sim)
			metrics := &spell.SpellMetrics[target.UnitIndex]
			advanceOneTick(sim, dot)
			first := metrics.TotalHealing
			caster.AddStatDynamic(sim, stats.HealingPower, 100)
			advanceOneTick(sim, dot)
			second := metrics.TotalHealing - first
			if first <= 0 {
				t.Fatalf("first tick healed %v", first)
			}
			if got := second - first; got != tc.gain {
				t.Errorf("tick after +100 healing power healed %v more than the first, want %v (ticks %v, %v)", got, tc.gain, first, second)
			}
		})
	}
}
