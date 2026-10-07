package rogue_test

import (
	"math"
	"testing"
)

// The tests in this file pin rogue talent and spell numbers to the client's
// own text (data/builds/1.60.1.70009/talents/rogue.json and
// spellconst/rogue.json). Each engine value was a vanilla-era figure that
// the Forever client changed.

func assertClose(t *testing.T, what string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// talentsWithRanks builds a talent string with several fields set.
func talentsWithRanks(t *testing.T, ranks map[string]int) string {
	t.Helper()
	str := zeroRogueTalents
	for name, rank := range ranks {
		str = talentStringWithRank(t, str, name, rank)
	}
	return str
}

// Improved Eviscerate: "Increases the damage done by your Eviscerate
// ability by 7%/13%/20%." (the engine had 5/10/15).
func TestImprovedEviscerateFollowsTheClientRanks(t *testing.T) {
	for rank, want := range map[int]float64{0: 1.0, 1: 1.07, 2: 1.13, 3: 1.20} {
		rogue := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"improved_eviscerate": rank}), "combat_backstab_prebis").GetRogue()
		assertClose(t, "Eviscerate.DamageMultiplier at rank "+string(rune('0'+rank)), rogue.Eviscerate.DamageMultiplier, want)
	}
}

// Lethality: "...critical strike damage bonus ... by 4%/8%/12%/16%/20%"
// (the engine had 6% per rank).
func TestLethalityIsFourPercentPerRank(t *testing.T) {
	rogue := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"lethality": 5}), "combat_backstab_prebis").GetRogue()
	// core.Spell stores the crit multiplier as 1 + the configured bonus.
	assertClose(t, "Backstab.CritDamageBonus", rogue.Backstab.CritDamageBonus, 1.20)
	assertClose(t, "SinisterStrike.CritDamageBonus", rogue.SinisterStrike.CritDamageBonus, 1.20)
}

// Opportunity: two ranks, "5%/10%" (Backstab, Garrote, Ambush, Mutilate).
// The engine indexed a five-rank 4%-per-rank table by the two-rank value.
func TestOpportunityIsFiveAndTenPercent(t *testing.T) {
	rogue := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"opportunity": 2}), "combat_backstab_prebis").GetRogue()
	assertClose(t, "Backstab.DamageMultiplier", rogue.Backstab.DamageMultiplier, 1.5*1.10)
	assertClose(t, "Ambush.DamageMultiplier", rogue.Ambush.DamageMultiplier, 2.5*1.10)
	assertClose(t, "Garrote.DamageMultiplier", rogue.Garrote.DamageMultiplier, 1.10)
}

// Dual Wield Specialization: "off-hand ... by 5%/10%/15%/20%/25%" (the
// engine had 10% per rank, so 50% at five).
func TestDualWieldSpecializationIsFivePercentPerRank(t *testing.T) {
	plain := buildRogueForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis").GetRogue()
	spec := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"dual_wield_specialization": 5}), "combat_backstab_prebis").GetRogue()
	assertClose(t, "off-hand damage multiplier ratio",
		spec.AutoAttacks.OHConfig().DamageMultiplier/plain.AutoAttacks.OHConfig().DamageMultiplier, 1.25)
}

// Vigor: "Increases your maximum Energy by 5." / "by 10." (the engine gave
// +10 at either rank).
func TestVigorAddsFiveEnergyPerRank(t *testing.T) {
	for rank, want := range map[int]float64{0: 100, 1: 105, 2: 110} {
		rogue := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"vigor": rank}), "combat_backstab_prebis").GetRogue()
		assertClose(t, "maximum energy at Vigor rank "+string(rune('0'+rank)), rogue.MaxEnergy(), want)
	}
}

// Hemorrhage: "100% weapon damage (145% if a Dagger is equipped)"; Ghostly
// Strike: "125% (180% if a Dagger is equipped) weapon damage". Backstab
// gear is daggers, Sinister Strike gear is swords.
func TestHemorrhageAndGhostlyStrikeScaleWithDaggers(t *testing.T) {
	talents := talentsWithRanks(t, map[string]int{"hemorrhage": 1, "ghostly_strike": 1})

	dagger := buildRogueForTalentTest(t, talents, "combat_backstab_prebis").GetRogue()
	assertClose(t, "dagger Hemorrhage.DamageMultiplier", dagger.Hemorrhage.DamageMultiplier, 1.45)
	assertClose(t, "dagger GhostlyStrike.DamageMultiplier", dagger.GhostlyStrike.DamageMultiplier, 1.80)

	sword := buildRogueForTalentTest(t, talents, "combat_sinister_strike_prebis").GetRogue()
	assertClose(t, "sword Hemorrhage.DamageMultiplier", sword.Hemorrhage.DamageMultiplier, 1.00)
	assertClose(t, "sword GhostlyStrike.DamageMultiplier", sword.GhostlyStrike.DamageMultiplier, 1.25)
}
