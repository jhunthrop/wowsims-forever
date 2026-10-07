package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

const (
	unstableAfflictionRank3SpellID = 1242972
	unstableAfflictionRank3Tick    = 174.0 // spellconst 1242972 effect 0: 1044 over 18 s in six ticks
	landAttempts                   = 20
)

// spellByID is the registered spell among spells with the given id.
func spellByID(t *testing.T, spells []*core.Spell, id int32) *core.Spell {
	t.Helper()
	for _, spell := range spells {
		if spell.ActionID.SpellID == id {
			return spell
		}
	}
	t.Fatalf("no registered spell with id %d", id)
	return nil
}

// TestUnstableAfflictionRegistersAllThreeRanksAtLevel60 checks the client's
// three rows, with the levels the client states for them (the client's rank 2,
// 1242971, is learned at 60 and its rank 3, 1242972, at 50).
func TestUnstableAfflictionRegistersAllThreeRanksAtLevel60(t *testing.T) {
	_, built, _ := newBareWarlockForDamageTest(t)

	want := []struct {
		id       int32
		rank     int
		level    int
		manaCost float64
	}{
		{427717, 1, 40, 200},
		{1242971, 2, 60, 265},
		{1242972, 3, 50, 340},
	}
	if len(built.UnstableAffliction) != len(want) {
		t.Fatalf("level-60 warlock registers %d Unstable Affliction ranks, want %d", len(built.UnstableAffliction), len(want))
	}
	for i, w := range want {
		spell := built.UnstableAffliction[i]
		if spell.ActionID.SpellID != w.id || spell.Rank != w.rank || spell.RequiredLevel != w.level {
			t.Errorf("rank slot %d: id %d rank %d level %d, want id %d rank %d level %d",
				i, spell.ActionID.SpellID, spell.Rank, spell.RequiredLevel, w.id, w.rank, w.level)
		}
		if got := spell.DefaultCast.CastTime.Milliseconds(); got != 1500 {
			t.Errorf("id %d cast time %d ms, want 1500", w.id, got)
		}
		if spell.ClientBaseDamage == [2]float64{} {
			t.Errorf("id %d does not declare ClientBaseDamage", w.id)
		}
	}
}

func TestUnstableAfflictionRank3IsSixTicksOfClientDamageOverEighteenSeconds(t *testing.T) {
	for attempt := 0; attempt < landAttempts; attempt++ {
		sim, built, target := newBareWarlockForDamageTest(t)
		spell := spellByID(t, built.UnstableAffliction, unstableAfflictionRank3SpellID)
		spell.ApplyEffects(sim, target, spell)

		dot := spell.Dot(target)
		if !dot.IsActive() {
			continue
		}
		if got := dot.SnapshotBaseDamage; got != unstableAfflictionRank3Tick {
			t.Errorf("per-tick snapshot damage = %.2f, want %.2f", got, unstableAfflictionRank3Tick)
		}
		if dot.NumberOfTicks != 6 || dot.TickLength.Seconds() != 3 {
			t.Errorf("dot is %d ticks of %v, want 6 ticks of 3s (18 s)", dot.NumberOfTicks, dot.TickLength)
		}
		return
	}
	t.Fatal("Unstable Affliction never landed")
}

// TestUnstableAfflictionAndImmolateExcludeEachOther is the live text: "Only one
// Unstable Affliction or Immolate per Warlock can be active on any one target."
func TestUnstableAfflictionAndImmolateExcludeEachOther(t *testing.T) {
	for attempt := 0; attempt < landAttempts; attempt++ {
		sim, built, target := newBareWarlockForDamageTest(t)
		immolate := built.Immolate[len(built.Immolate)-1]
		unstable := spellByID(t, built.UnstableAffliction, unstableAfflictionRank3SpellID)

		immolate.ApplyEffects(sim, target, immolate)
		if !immolate.Dot(target).IsActive() {
			continue
		}
		unstable.ApplyEffects(sim, target, unstable)
		if !unstable.Dot(target).IsActive() {
			continue
		}
		if immolate.Dot(target).IsActive() {
			t.Fatal("Unstable Affliction landed but the target still carries Immolate")
		}

		immolate.ApplyEffects(sim, target, immolate)
		if immolate.Dot(target).IsActive() && unstable.Dot(target).IsActive() {
			t.Fatal("Immolate landed but the target still carries Unstable Affliction")
		}
		if immolate.Dot(target).IsActive() {
			return
		}
	}
	t.Fatal("the two dots never both landed in turn")
}

// TestUnstableAfflictionIgnoresOtherAfflictionDots checks the exclusion is only
// with Immolate: Corruption and Unstable Affliction share a target.
func TestUnstableAfflictionIgnoresOtherAfflictionDots(t *testing.T) {
	for attempt := 0; attempt < landAttempts; attempt++ {
		sim, built, target := newBareWarlockForDamageTest(t)
		corruption := built.Corruption[len(built.Corruption)-1]
		unstable := spellByID(t, built.UnstableAffliction, unstableAfflictionRank3SpellID)

		corruption.ApplyEffects(sim, target, corruption)
		unstable.ApplyEffects(sim, target, unstable)
		if corruption.Dot(target).IsActive() && unstable.Dot(target).IsActive() {
			return
		}
	}
	t.Fatal("Corruption and Unstable Affliction never coexisted")
}
