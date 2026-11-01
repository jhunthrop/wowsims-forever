package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's DoTs read the caster's spell power and damage multipliers at
// every tick (forever-bugs #137, Blizzard 2026-10-01: "Since we've changed
// to dynamic DoTs"; forever-bugs #398, a combat log of Bane of Agony and
// Blizzard 2026-10-26: "adjust every tick to the spell power of the
// caster"). A DotConfig marked Dynamic re-takes its snapshot before each
// tick, so spell power gained after the cast reaches the ticks still to
// come; a snapshotting dot keeps what it had when it was applied.
func TestDynamicDotReadsSpellPowerEachTick(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dynamic bool
		gain    float64
	}{
		{"dynamic", true, 100},
		{"snapshot", false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, caster, target := newPeriodicCritFixture(t, 0)
			spell := caster.RegisterSpell(SpellConfig{
				ActionID:         ActionID{SpellID: 10894},
				SpellSchool:      SpellSchoolShadow,
				DefenseType:      DefenseTypeMagic,
				Flags:            SpellFlagPureDot,
				ProcMask:         ProcMaskSpellDamage,
				DamageMultiplier: 1,
				ThreatMultiplier: 1,
				Dot: DotConfig{
					Aura:             Aura{Label: "Test Dynamic Dot"},
					NumberOfTicks:    2,
					TickLength:       time.Second * 3,
					BonusCoefficient: 1,
					Dynamic:          tc.dynamic,
					OnSnapshot: func(_ *Simulation, target *Unit, dot *Dot, isRollover bool) {
						dot.Snapshot(target, 100, isRollover)
					},
					OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
						dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
					},
				},
			})
			advanceFixture(sim, spell.Dot(target))
			dot := spell.Dot(target)
			dot.Apply(sim)
			metrics := &spell.SpellMetrics[target.UnitIndex]
			advanceOneTick(sim, dot)
			first := metrics.TotalTickDamage
			caster.AddStatDynamic(sim, stats.SpellPower, 100)
			advanceOneTick(sim, dot)
			second := metrics.TotalTickDamage - first
			if first <= 0 {
				t.Fatalf("first tick dealt %v", first)
			}
			if got := second - first; got != tc.gain {
				t.Errorf("tick after +100 spell power dealt %v more than the first, want %v (ticks %v, %v)", got, tc.gain, first, second)
			}
		})
	}
}

// advanceFixture finishes the fixture's deferred finalize without ticking.
func advanceFixture(sim *Simulation, dot *Dot) {
	if finish, ok := pendingFixtureFinalize[sim]; ok {
		delete(pendingFixtureFinalize, sim)
		finish()
		caster := dot.Spell.Unit
		caster.Rotation = &APLRotation{unit: caster}
	}
}
