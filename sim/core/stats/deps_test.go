package stats

import (
	"testing"
)

func TestStatDependencies(t *testing.T) {
	baseStat := Stats{
		Stamina:   1,
		Intellect: 1,
		Agility:   2,
		Spirit:    1,
	}

	sdm := NewStatDependencyManager()

	sdm.MultiplyStat(Intellect, 2)
	sdm.AddStatDependency(Stamina, Intellect, 1)
	sdm.MultiplyStat(Stamina, 2)
	sdm.AddStatDependency(Agility, Stamina, 1)

	dep1 := sdm.NewDynamicMultiplyStat(Agility, 2)
	dep2 := sdm.NewDynamicMultiplyStat(Spirit, 3)
	dep3 := sdm.NewDynamicStatDependency(Agility, Spirit, 0.75)

	sdm.FinalizeStatDeps()

	result := sdm.ApplyStatDependencies(baseStat)
	expectedResult := Stats{
		Stamina:   6,
		Intellect: 14,
		Agility:   2,
		Spirit:    1,
	}
	if !result.Equals(expectedResult) {
		t.Fatalf("Stats do not match:\nActual: %s\nExpected: %s", result, expectedResult)
	}

	sdm.EnableDynamicStatDep(dep1)
	result2 := sdm.ApplyStatDependencies(baseStat)
	expectedResult2 := Stats{
		Stamina:   10,
		Intellect: 22,
		Agility:   4,
		Spirit:    1,
	}
	if !result2.Equals(expectedResult2) {
		t.Fatalf("Updated stats do not match:\nActual: %s\nExpected: %s", result2, expectedResult2)
	}

	sdm.EnableDynamicStatDep(dep2)
	result3 := sdm.ApplyStatDependencies(baseStat)
	expectedResult3 := Stats{
		Stamina:   10,
		Intellect: 22,
		Agility:   4,
		Spirit:    3,
	}
	if !result3.Equals(expectedResult3) {
		t.Fatalf("Updated stats do not match:\nActual: %s\nExpected: %s", result3, expectedResult3)
	}

	sdm.DisableDynamicStatDep(dep2)
	result4 := sdm.ApplyStatDependencies(baseStat)
	if !result4.Equals(expectedResult2) {
		t.Fatalf("Updated stats do not match:\nActual: %s\nExpected: %s", result4, expectedResult2)
	}

	sdm.EnableDynamicStatDep(dep3)
	result5 := sdm.ApplyStatDependencies(baseStat)
	expectedResult5 := Stats{
		Stamina:   10,
		Intellect: 22,
		Agility:   4,
		Spirit:    4,
	}
	if !result5.Equals(expectedResult5) {
		t.Fatalf("Updated stats do not match:\nActual: %s\nExpected: %s", result5, expectedResult5)
	}

	sdm.DisableDynamicStatDep(dep3)
	result6 := sdm.ApplyStatDependencies(baseStat)
	if !result6.Equals(expectedResult2) {
		t.Fatalf("Updated stats do not match:\nActual: %s\nExpected: %s", result6, expectedResult2)
	}
}

func TestMultipleStatDep(t *testing.T) {
	sdm := NewStatDependencyManager()

	baseStat := Stats{
		Intellect:  100,
		SpellPower: 100,
	}

	sdm.AddStatDependency(Intellect, SpellPower, 0.2)
	sdm.AddStatDependency(Intellect, SpellPower, 0.2)
	sdm.MultiplyStat(Intellect, 1.2)
	sdm.FinalizeStatDeps()
	result := sdm.ApplyStatDependencies(baseStat)

	expectedResult := Stats{
		Intellect:  100 * 1.2,
		SpellPower: 100 + (100*1.2)*(0.2+0.2),
	}

	if !result.Equals(expectedResult) {
		t.Fatalf("Stats do not match:\nActual: %s\nExpected: %s", result, expectedResult)
	}
}

// Forever: bonus healing carries one third as bonus damage. Dependency
// evaluation is single-pass over safeDepsOrder, so the source must appear
// before the destination or AddStatDependency panics. Nothing depends on
// HealingPower, so moving it above SpellDamage is safe — and this test is
// what stops a later reorder from silently breaking it.
func TestHealingPowerCanFeedSpellDamage(t *testing.T) {
	if !isValidDep(HealingPower, SpellDamage) {
		t.Fatal("HealingPower --> SpellDamage is not a valid dependency; " +
			"HealingPower must appear before SpellDamage in safeDepsOrder")
	}
}

func TestHealingToSpellDamageAppliesOneThird(t *testing.T) {
	sdm := NewStatDependencyManager()
	sdm.AddStatDependency(HealingPower, SpellDamage, 1.0/3.0)
	sdm.FinalizeStatDeps()

	got := sdm.ApplyStatDependencies(Stats{HealingPower: 300, SpellDamage: 50})
	if got[HealingPower] != 300 {
		t.Errorf("HealingPower = %v, want it unchanged at 300", got[HealingPower])
	}
	if got[SpellDamage] != 150 {
		t.Errorf("SpellDamage = %v, want 150 (50 base + 300/3)", got[SpellDamage])
	}
}
