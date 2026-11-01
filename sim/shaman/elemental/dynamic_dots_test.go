package elemental

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's DoTs read spell power and the caster's multipliers at every
// tick (forever-bugs #137, Blizzard 2026-10-01: "Since we've changed to
// dynamic DoTs"; #398, 2026-10-26).
func TestFlameShockReadsSpellPowerEachTick(t *testing.T) {
	sim, built := newCallOfFlameShaman(t, "")
	target := sim.Encounter.TargetUnits[0]
	spell := built.FlameShock[len(built.FlameShock)-1]
	for attempt := 0; attempt < 20 && !spell.Dot(target).IsActive(); attempt++ {
		spell.ApplyEffects(sim, target, spell)
	}
	dot := spell.Dot(target)
	if !dot.IsActive() {
		t.Fatal("Flame Shock never landed")
	}
	dot.TickCount = 1
	dot.TickOnce(sim)
	before := dot.SnapshotBaseDamage
	built.AddStatDynamic(sim, stats.SpellPower, 100)
	dot.TickCount = 2
	dot.TickOnce(sim)
	if want := before + 100*dot.BonusCoefficient; math.Abs(dot.SnapshotBaseDamage-want) > 1e-9 {
		t.Errorf("tick 2 base %v after +100 spell power, want %v", dot.SnapshotBaseDamage, want)
	}
}
