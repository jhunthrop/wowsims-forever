package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's DoTs read spell power and the caster's multipliers at every
// tick: forever-bugs #137 (Blizzard, 2026-10-01: "Since we've changed to
// dynamic DoTs"), #35 (Eureka! boosts DoTs already ticking) and #398
// (Blizzard, 2026-10-26: Bane of Agony gained "scripting to adjust every
// tick to the spell power of the caster", still snapshotting Amplify Curse).
const dynamicDotsTalents = "25520100135001050-0005200000000000000-0550004000000000"

// landedDot casts spell until its dot lands and returns it.
func landedDot(t *testing.T, sim *core.Simulation, spell *core.Spell, target *core.Unit) *core.Dot {
	t.Helper()
	for attempt := 0; attempt < landAttempts; attempt++ {
		spell.ApplyEffects(sim, target, spell)
		if dot := spell.Dot(target); dot.IsActive() {
			return dot
		}
	}
	t.Fatalf("%v never landed", spell.ActionID)
	return nil
}

// tickTo runs the dot's tick number n (1-based) at the current time.
func tickTo(dot *core.Dot, n int32, sim *core.Simulation) {
	dot.TickCount = n
	dot.TickOnce(sim)
}

func TestWarlockDotsReadSpellPowerEachTick(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spell func(*Warlock) *core.Spell
	}{
		{"Corruption", func(w *Warlock) *core.Spell { return w.Corruption[len(w.Corruption)-1] }},
		{"Immolate", func(w *Warlock) *core.Spell { return w.Immolate[len(w.Immolate)-1] }},
		{"Siphon Life", func(w *Warlock) *core.Spell { return w.SiphonLife[len(w.SiphonLife)-1] }},
		{"Bane of Agony", func(w *Warlock) *core.Spell { return w.CurseOfAgony[len(w.CurseOfAgony)-1] }},
		{"Bane of Doom", func(w *Warlock) *core.Spell { return w.CurseOfDoom }},
		{"Drain Life", func(w *Warlock) *core.Spell { return w.DrainLife[len(w.DrainLife)-1] }},
		{"Drain Soul", func(w *Warlock) *core.Spell { return w.DrainSoul[len(w.DrainSoul)-1] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, built, target := newWarlockForDamageTest(t, dynamicDotsTalents)
			spell := tc.spell(built)
			dot := landedDot(t, sim, spell, target)
			tickTo(dot, 1, sim)
			before := dot.SnapshotBaseDamage
			multiplier := dot.SnapshotAttackerMultiplier

			built.AddStatDynamic(sim, stats.SpellPower, 100)
			tickTo(dot, 2, sim)

			want := before + 100*dot.BonusCoefficient
			if math.Abs(dot.SnapshotBaseDamage-want) > 1e-9 {
				t.Errorf("tick 2 base %v after +100 spell power, want %v (tick 1 %v, coefficient %v)", dot.SnapshotBaseDamage, want, before, dot.BonusCoefficient)
			}
			if math.Abs(dot.SnapshotAttackerMultiplier-multiplier) > 1e-9 {
				t.Errorf("tick 2 multiplier %v, want tick 1's %v (a per-dot multiplier was lost)", dot.SnapshotAttackerMultiplier, multiplier)
			}
		})
	}
}

// Bane of Agony keeps its ramp under the per-tick read: half of the base
// for ticks 1-4, all of it for 5-8 and half again for 9-12, with the spell
// power the caster has at each tick added in full.
func TestBaneOfAgonyRampsUnderTheDynamicRead(t *testing.T) {
	sim, built, target := newWarlockForDamageTest(t, "")
	spell := built.CurseOfAgony[len(built.CurseOfAgony)-1]
	dot := landedDot(t, sim, spell, target)
	base := CurseOfAgonyTickDamage[len(CurseOfAgonyTickDamage)-1].Center(int(built.Level)) * spell.BaseDamageMultiplierAdditive

	for tick := int32(1); tick <= 12; tick++ {
		if tick == 7 {
			built.AddStatDynamic(sim, stats.SpellPower, 100)
		}
		tickTo(dot, tick, sim)
		stage := 0.5 * float64(1+(tick-1)/4)
		want := base*stage + dot.BonusCoefficient*spell.GetBonusDamage(target)
		if math.Abs(dot.SnapshotBaseDamage-want) > 1e-9 {
			t.Errorf("tick %d base %v, want %v (stage %v)", tick, dot.SnapshotBaseDamage, want, stage)
		}
	}
}
