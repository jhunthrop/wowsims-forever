package holy

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	clientPaladinSpellconst = "../../core/testdata/conformance/client/paladin.json"

	// healingPower is what the numeric tests give the healer, so the
	// coefficient is part of what they check.
	healingPower = 500.0
	// spellCritMultiplier is the engine's magic crit multiplier (1.5).
	spellCritMultiplier = 1.5

	// averageTolerance is how far a sample average may sit from its
	// expectation (relative): the roll is uniform, a few thousand samples.
	averageTolerance = 0.006
	critTolerance    = 0.03
	numericRuns      = 200
	numericSeconds   = 60
)

func clientSpell(t *testing.T, id int32) spellconst.Spell {
	t.Helper()
	class, err := spellconst.Load(clientPaladinSpellconst)
	if err != nil {
		t.Fatalf("loading the client table: %v", err)
	}
	spell, ok := class.ByID(id)
	if !ok {
		t.Fatalf("spell %d is not in the client table", id)
	}
	return spell
}

func within(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance*want
}

// assertHealNumbers casts castID on the tank until the run is over and
// checks the heal recorded under healID against the client's roll at the
// healer's level plus the effect's coefficient times healing power.
func assertHealNumbers(t *testing.T, f fight, castID, healID int32) spellTotals {
	t.Helper()
	heal := clientSpell(t, healID)
	level := f.level
	if level == 0 {
		level = defaultLevel
	}
	low, high, ok := heal.DamageRange(0, int(level))
	if !ok {
		t.Fatalf("spell %d has no effect 0", healID)
	}
	want := (low+high)/2 + heal.Effects[0].ResolvedSPCoefficient*healingPower

	totals := totalsOf(f.run(t, numericRuns), healID)
	if totals.landed() == 0 {
		t.Fatalf("spell %d never landed a heal", healID)
	}
	if got := totals.normalHeal(); !within(got, want, averageTolerance) {
		t.Errorf("spell %d at level %d: normal heal averages %.1f, want %.1f (roll %.1f-%.1f + %.3f x %.0f)",
			healID, level, got, want, low, high, heal.Effects[0].ResolvedSPCoefficient, healingPower)
	}
	if totals.crits > 0 {
		if got := totals.critHeal(); !within(got, want*spellCritMultiplier, critTolerance) {
			t.Errorf("spell %d: crit heal averages %.1f, want %.1f", healID, got, want*spellCritMultiplier)
		}
	}
	return totals
}

func numericFight(rotation string) fight {
	return fight{
		rotation: rotation,
		duration: numericSeconds,
		bonus:    stats.Stats{stats.HealingPower: healingPower},
	}
}

var (
	holyLightRankIDs     = []int32{635, 639, 647, 1026, 1042, 3472, 10328, 10329, 25292}
	flashOfLightRankIDs  = []int32{19750, 19939, 19940, 19941, 19942, 19943}
	holyShockCastIDs     = []int32{1311606, 20473, 20929, 20930}
	holyShockHealIDs     = []int32{1311605, 25914, 25913, 25903}
	lightsVigilCastIDs   = []int32{1310911, 1311590, 1311595}
	lightsVigilHealIDs   = []int32{1310912, 1311591, 1311596}
	holyShockTalentRanks = map[string]int{"holy_shock": 1}
	lightsVigilTalent    = map[string]int{"lights_vigil": 1}
)

func TestEveryHolyLightRankHealsTheClientAmount(t *testing.T) {
	for _, id := range holyLightRankIDs {
		assertHealNumbers(t, numericFight(castLoop(id)), id, id)
	}
}

func TestEveryFlashOfLightRankHealsTheClientAmount(t *testing.T) {
	for _, id := range flashOfLightRankIDs {
		assertHealNumbers(t, numericFight(castLoop(id)), id, id)
	}
}

func TestEveryHolyShockRankHealsTheClientAmount(t *testing.T) {
	for i, cast := range holyShockCastIDs {
		f := numericFight(castLoop(cast))
		f.talents = holyShockTalentRanks
		assertHealNumbers(t, f, cast, holyShockHealIDs[i])
	}
}
