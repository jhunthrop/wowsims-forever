package warlock

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Affliction field 12 is Nightfall (2 ranks), field 14 Siphon Life, field 16
// Shadow Mastery and field 17 Wrack.
const (
	wrackNightfallTalents = "00000000000" + "2" + "0000" + "1"
	wrackSiphonTalents    = "0000000000000" + "1" + "00" + "1"
)

// TestWrackIsAChannel pins the client row: SpellMisc Attributes_1 for 1316697
// is 0x4004, the same channel bits Drain Life (11700) and Mind Flay (15407)
// carry, and Wowhead's Forever tooltip reads "Channeled (6 sec cast)". The
// warlock is busy for the channel, so nothing else is cast over Wrack's ticks.
func TestWrackIsAChannel(t *testing.T) {
	sim, warlock, target := newWarlockForDamageTest(t, WrackOnlyTalentsString)

	if !warlock.Wrack.Flags.Matches(core.SpellFlagChanneled) {
		t.Fatal("Wrack is not flagged as a channel")
	}
	warlock.Wrack.Cast(sim, target)
	if !warlock.Wrack.Dot(target).IsActive() {
		t.Fatal("Wrack's channel did not start")
	}
	if !warlock.IsChanneling(sim) {
		t.Error("the warlock is free to cast during Wrack's channel")
	}
}

// TestWrackNoLongerAmplifiesTheWarlocksDoTs pins the 10 Nov hotfix ("Wrack
// no longer increases the damage of the Warlock's periodic effects on the
// target"): push 113766 turns 1316697 effect 1 from aura 271 (damage taken
// from the caster, 10, class mask 0x402) into a dummy at 100, the speed-up
// below. No spell of the warlock's takes more damage while Wrack runs.
func TestWrackNoLongerAmplifiesTheWarlocksDoTs(t *testing.T) {
	sim, warlock, target := newWarlockForDamageTest(t, wrackSiphonTalents)

	last := func(spells []*core.Spell) *core.Spell { return spells[len(spells)-1] }
	spells := map[string]*core.Spell{
		"Corruption":     last(warlock.Corruption),
		"Bane of Agony":  last(warlock.CurseOfAgony),
		"Shadow Bolt":    last(warlock.ShadowBolt),
		"Siphon Life":    last(warlock.SiphonLife),
		"Wrack":          warlock.Wrack,
		"Bane of Doom":   warlock.CurseOfDoom,
		"Immolate (dot)": last(warlock.Immolate),
	}
	multiplier := func(spell *core.Spell) float64 {
		return spell.TargetDamageMultiplier(warlock.AttackTables[target.UnitIndex][spell.CastType], true)
	}
	before := map[string]float64{}
	for name, spell := range spells {
		before[name] = multiplier(spell)
	}
	warlock.Wrack.Dot(target).Apply(sim)
	for name, spell := range spells {
		if ratio := multiplier(spell) / before[name]; math.Abs(ratio-1) > 1e-9 {
			t.Errorf("%s: Wrack multiplies it by %.4f, want 1", name, ratio)
		}
	}
}

// dotTickTimes applies dot at the sim's current time, runs the given
// actions at their offsets from then, and returns the offset of each of the
// dot's ticks until it falls off (or until 30 s).
func dotTickTimes(t *testing.T, sim *core.Simulation, dot *core.Dot, at map[time.Duration]func(*core.Simulation)) []time.Duration {
	t.Helper()
	start := sim.CurrentTime
	dot.Apply(sim)
	for offset, action := range at {
		core.StartDelayedAction(sim, core.DelayedActionOptions{DoAt: start + offset, OnAction: action})
	}
	var ticks []time.Duration
	seen := dot.TickCount
	for dot.IsActive() && sim.CurrentTime < start+30*time.Second {
		if done := sim.Step(); done {
			break
		}
		if dot.TickCount != seen {
			seen = dot.TickCount
			ticks = append(ticks, sim.CurrentTime-start)
		}
	}
	if dot.IsActive() {
		t.Fatalf("%v still active 30 s after it was applied", dot.Spell.ActionID)
	}
	return ticks
}

func seconds(values ...float64) []time.Duration {
	out := make([]time.Duration, len(values))
	for i, v := range values {
		out[i] = time.Duration(v * float64(time.Second))
	}
	return out
}

// TestWrackChannelRunsCorruptionsClockTwiceAsFast pins the 10 Nov hotfix:
// "While channeling, your Siphon Life, Corruption, and Bane of Agony effects
// on the target deal their periodic damage 100% faster", and "the time it
// skips is kept after Wrack's channel ends". Corruption rank 6 ticks every
// 3 s for 18 s. With a 6 s Wrack from 1 s, its clock runs 12 s in those 6:
// the ticks due at 3, 6, 9 and 12 s of its own time land at 2, 3.5, 5 and
// 6.5 s; at 7 s it is 13 s into its run, so the last two land at 9 and 12 s
// and it is gone 6 s early.
func TestWrackChannelRunsCorruptionsClockTwiceAsFast(t *testing.T) {
	sim, warlock, target := newWarlockForDamageTest(t, wrackSiphonTalents)
	corruption := warlock.Corruption[len(warlock.Corruption)-1].Dot(target)
	wrack := warlock.Wrack.Dot(target)

	got := dotTickTimes(t, sim, corruption, map[time.Duration]func(*core.Simulation){
		time.Second: func(sim *core.Simulation) { wrack.Apply(sim) },
	})
	if want := seconds(2, 3.5, 5, 6.5, 9, 12); !slices.Equal(got, want) {
		t.Errorf("Corruption ticked at %v, want %v", got, want)
	}
}

// TestWrackKeepsTheTimeItSkipped is the hotfix note's own example: "if you
// channel Wrack for 1 second during a 3-second Corruption tick, Corruption
// advances 2 seconds instead of 1, so its next tick lands 1 second later
// instead of 2". Wrack starts with Corruption and stops after 1 s.
func TestWrackKeepsTheTimeItSkipped(t *testing.T) {
	sim, warlock, target := newWarlockForDamageTest(t, wrackSiphonTalents)
	corruption := warlock.Corruption[len(warlock.Corruption)-1].Dot(target)
	wrack := warlock.Wrack.Dot(target)

	got := dotTickTimes(t, sim, corruption, map[time.Duration]func(*core.Simulation){
		0:           func(sim *core.Simulation) { wrack.Apply(sim) },
		time.Second: func(sim *core.Simulation) { wrack.Cancel(sim) },
	})
	if want := seconds(2, 5, 8, 11, 14, 17); !slices.Equal(got, want) {
		t.Errorf("Corruption ticked at %v, want %v", got, want)
	}
}

// TestWrackSpeedsUpSiphonLifeAndAgonyButNotDoom: the hotfix names Siphon
// Life, Corruption and Bane of Agony. Bane of Doom's one tick and
// Immolate's burn keep their own clocks.
func TestWrackSpeedsUpSiphonLifeAndAgonyButNotDoom(t *testing.T) {
	last := func(spells []*core.Spell) *core.Spell { return spells[len(spells)-1] }
	for _, c := range []struct {
		name    string
		spell   func(*Warlock) *core.Spell
		hastens bool
	}{
		{"Siphon Life", func(w *Warlock) *core.Spell { return last(w.SiphonLife) }, true},
		{"Bane of Agony", func(w *Warlock) *core.Spell { return last(w.CurseOfAgony) }, true},
		{"Bane of Doom", func(w *Warlock) *core.Spell { return w.CurseOfDoom }, false},
		{"Immolate", func(w *Warlock) *core.Spell { return last(w.Immolate) }, false},
	} {
		sim, warlock, target := newWarlockForDamageTest(t, wrackSiphonTalents)
		dot := c.spell(warlock).Dot(target)
		period := dot.TickPeriod()
		got := dotTickTimes(t, sim, dot, map[time.Duration]func(*core.Simulation){
			0: func(sim *core.Simulation) { warlock.Wrack.Dot(target).Apply(sim) },
		})
		want := period
		if c.hastens {
			want = period / 2
		}
		if len(got) == 0 || got[0] != want {
			t.Errorf("%s: first tick at %v with Wrack channelled from its start, want %v", c.name, got, want)
		}
	}
}

// Affliction field 15 is Soul Siphon (3 ranks).
const wrackSoulSiphonTalents = "0000000000000" + "1" + "3" + "0" + "1"

// TestSoulSiphonCountsCorruptionSiphonLifeAndAgony pins the 10 Nov hotfix:
// Soul Siphon gives Drain Life, Drain Soul and Wrack "4/8/12% per each of
// your Corruption, Siphon Life, and Bane of Agony spells active on the
// target, up to a maximum of 36% (was any Affliction spell)". Bane of Doom
// no longer counts.
func TestSoulSiphonCountsCorruptionSiphonLifeAndAgony(t *testing.T) {
	sim, warlock, target := newWarlockForDamageTest(t, wrackSoulSiphonTalents)
	last := func(spells []*core.Spell) *core.Spell { return spells[len(spells)-1] }

	steps := []struct {
		apply *core.Spell
		want  float64
	}{
		{warlock.CurseOfDoom, 1},
		{last(warlock.Corruption), 1.12},
		{last(warlock.SiphonLife), 1.24},
		{last(warlock.CurseOfAgony), 1.36},
	}
	for _, step := range steps {
		step.apply.Dot(target).Apply(sim)
		if got := warlock.soulSiphonMultiplier(target, warlock.Wrack); !floatsNearlyEqual(got, step.want) {
			t.Errorf("Wrack's Soul Siphon multiplier after %v = %.2f, want %.2f", step.apply.ActionID, got, step.want)
		}
	}
	warlock.Wrack.Dot(target).Apply(sim)
	if got := warlock.soulSiphonMultiplier(target, last(warlock.DrainLife)); !floatsNearlyEqual(got, 1.36) {
		t.Errorf("Drain Life's Soul Siphon multiplier with Wrack up = %.2f, want 1.36 (Wrack does not count)", got)
	}
}

// TestNightfallProcsFromWrackTicks pins the talent text (18094, 70707):
// "Gives your Corruption, Drain Soul, Drain Life, and Wrack spells a 4% chance
// to cause you to enter a Shadow Trance after damaging the opponent." The
// 2026-10-07 deep dive lists Drain Soul and Wrack as new triggers.
func TestNightfallProcsFromWrackTicks(t *testing.T) {
	sim, warlock, target := newWarlockForDamageTest(t, wrackNightfallTalents)
	trance := warlock.GetAura("Nightfall Shadow Trance")
	if trance == nil {
		t.Fatal("no Shadow Trance aura with Nightfall 2/2")
	}

	// The ticks run through the dot's own tick handler on a snapshot, without
	// starting the channel, which would need a rotation to end.
	dot := warlock.Wrack.Dot(target)
	dot.TakeSnapshot(sim, false)
	procs := 0
	for tick := 0; tick < 1200; tick++ {
		dot.OnTick(sim, target, dot)
		if trance.IsActive() {
			procs++
			trance.Deactivate(sim)
		}
	}
	// 1200 ticks at 4%: 48 expected.
	if procs < 20 || procs > 90 {
		t.Errorf("Wrack ticks gave %d Shadow Trances over 1200 ticks, want about 48", procs)
	}
}

// TestNightfallTriggersAreTheTalentText keeps the trigger list to the four
// spells the text names.
func TestNightfallTriggersAreTheTalentText(t *testing.T) {
	want := []int32{SpellCode_WarlockCorruption, SpellCode_WarlockDrainSoul, SpellCode_WarlockDrainLife, SpellCode_WarlockWrack}
	if len(nightfallTriggerSpellCodes) != len(want) {
		t.Fatalf("Nightfall triggers %v, want %v", nightfallTriggerSpellCodes, want)
	}
	for i := range want {
		if nightfallTriggerSpellCodes[i] != want[i] {
			t.Fatalf("Nightfall triggers %v, want %v", nightfallTriggerSpellCodes, want)
		}
	}
}
