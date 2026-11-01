package warlock

import (
	"math"
	"testing"

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

// TestWrackVulnerabilityCoversOnlyCorruptionAndAgony pins effect 1 of 1316697:
// aura 271 (damage taken from the caster) at 10 on class mask 0x402, which is
// Corruption (0x2) and Bane of Agony (0x400) only. Shadow Bolt, Siphon Life
// and Wrack itself take nothing from it.
func TestWrackVulnerabilityCoversOnlyCorruptionAndAgony(t *testing.T) {
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
	want := map[string]float64{"Corruption": 1.10, "Bane of Agony": 1.10}

	multiplier := func(spell *core.Spell) float64 {
		return spell.TargetDamageMultiplier(warlock.AttackTables[target.UnitIndex][spell.CastType], true)
	}
	before := map[string]float64{}
	for name, spell := range spells {
		before[name] = multiplier(spell)
	}
	warlock.Wrack.ApplyEffects(sim, target, warlock.Wrack)
	if !warlock.WrackVulnerabilityAuras.Get(target).IsActive() {
		t.Fatal("Wrack landed but its vulnerability is not up")
	}
	for name, spell := range spells {
		ratio := multiplier(spell) / before[name]
		expected := want[name]
		if expected == 0 {
			expected = 1
		}
		if math.Abs(ratio-expected) > 1e-9 {
			t.Errorf("%s: Wrack's vulnerability multiplies it by %.4f, want %.2f", name, ratio, expected)
		}
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
