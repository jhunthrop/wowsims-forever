package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// N2 (Task 4 re-review 1): the review's H1 finding was seven effects that
// granted a melee value and a spell value under the old split stats and
// were rewritten as two consecutive writes of the merged stat instead of
// one, doubling every affected buff and talent. That was only caught by
// eyeballing the .results deltas. These tests apply two of the fixed
// buffs — Rallying Cry of the Dragonslayer and Songflower Serenade — to a
// bare unit and assert the resulting Crit is exactly the larger of the
// two pre-merge values, applied once, so a re-introduced second write
// fails here instead of only moving goldens.

func newBareTestUnit() *Unit {
	return &Unit{
		Type:        PlayerUnit,
		Index:       0,
		Level:       60,
		auraTracker: newAuraTracker(),
		Env:         &Environment{MeasuringStats: true},
	}
}

func TestRallyingCryOfTheDragonslayerAppliesCritOnce(t *testing.T) {
	unit := newBareTestUnit()
	ApplyRallyingCryOfTheDragonslayer(unit, "TestRallyingCry")

	aura := unit.GetAura("Rallying Cry of the Dragonslayer")
	if aura == nil {
		t.Fatal("Rallying Cry of the Dragonslayer aura was not registered")
	}
	aura.Activate(&Simulation{})

	// Forever merges the pre-merge SpellCrit 10 and MeleeCrit 5 into one
	// Crit write at the larger value. A re-introduced second write would
	// make this 15 (10 + 5).
	wantCrit := 10 * float64(CritRatingPerCritChance)
	if got := unit.GetStat(stats.Crit); got != wantCrit {
		t.Fatalf("Crit = %v, want %v (max of pre-merge SpellCrit 10 and MeleeCrit 5, applied once)", got, wantCrit)
	}
	// Unaffected stats confirm the aura actually activated rather than the
	// test silently checking a no-op.
	if got := unit.GetStat(stats.AttackPower); got != 140 {
		t.Fatalf("AttackPower = %v, want 140", got)
	}
}

func TestSongflowerSerenadeAppliesCritOnce(t *testing.T) {
	unit := newBareTestUnit()
	ApplySongflowerSerenade(unit)

	aura := unit.GetAura("Songflower Serenade")
	if aura == nil {
		t.Fatal("Songflower Serenade aura was not registered")
	}
	aura.Activate(&Simulation{})

	// Forever merges the pre-merge MeleeCrit 5 and SpellCrit 5 into one
	// Crit write at the larger (here, equal) value. A re-introduced second
	// write would make this 10 (5 + 5).
	if got := unit.GetStat(stats.Crit); got != 5 {
		t.Fatalf("Crit = %v, want 5 (max of pre-merge MeleeCrit 5 and SpellCrit 5, applied once)", got)
	}
	if got := unit.GetStat(stats.Agility); got != 15 {
		t.Fatalf("Agility = %v, want 15", got)
	}
}
