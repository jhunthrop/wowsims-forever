package balance

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/druid"
)

// manaRestoredByWraths hits the target with the highest Wrath rank n times
// from empty mana and returns the mana the druid ends with, regeneration
// included.
func manaRestoredByWraths(t *testing.T, relicID int32, n int) float64 {
	t.Helper()
	built, sim, target := newBalanceDruidSimWearing(t, druidTalentsString(t, nil), core.RelicEquipment(relicID))
	built.SpendMana(sim, built.CurrentMana(), built.NewManaMetrics(core.ActionID{SpellID: 1}))
	wrath := built.Wrath[len(built.Wrath)-1]
	for i := 0; i < n; i++ {
		wrath.ApplyEffects(sim, target, wrath.Spell)
	}
	// Wrath's damage lands after its missile travels; let every one arrive.
	for !sim.Step() && sim.CurrentTime < 10*time.Second {
	}
	return built.CurrentMana()
}

// Talons of Wrath (client spell 1248996, mana from spell 1302521): Wrath has
// a 50% chance to restore 35 mana. Both fights draw every other roll from
// the same streams and regenerate alike, so the mana the idol adds is the
// difference between them.
func TestTalonsOfWrathRestoresManaOnWrathHits(t *testing.T) {
	const wraths = 40 // few enough that the druid's mana never reaches its cap
	restored := manaRestoredByWraths(t, druid.TalonsOfWrath, wraths) - manaRestoredByWraths(t, 0, wraths)
	procs := restored / 35
	if restored <= 0 || math.Abs(procs-math.Round(procs)) > 1e-6 {
		t.Fatalf("the idol added %v mana over %d Wraths, want a positive whole number of 35s", restored, wraths)
	}
	// A 50% proc over 40 hits: 20 expected, so 5..35 is far past noise and
	// far from either "never" or "always".
	if procs < 5 || procs > 35 {
		t.Errorf("%v procs in %d Wraths, want about half", procs, wraths)
	}
}
