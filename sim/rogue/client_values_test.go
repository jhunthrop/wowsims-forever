package rogue_test

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/rogue"
)

// The tests in this file pin rogue talent and spell numbers to the client's
// own data: talent ranks from the 70755 trait curves
// (TraitDefinitionEffectPoints -> CurvePoint), spell values from the rows.

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

// Coup de Grace (14162, node 105708): e0 aura 108 on Eviscerate, curve
// 83054: 7/13/20%.
func TestCoupDeGraceFollowsTheClientCurve(t *testing.T) {
	for rank, want := range map[int]float64{0: 1.0, 1: 1.07, 2: 1.13, 3: 1.20} {
		rogue := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"coup_de_grace": rank}), "combat_backstab_prebis").GetRogue()
		assertClose(t, "Eviscerate.DamageMultiplier at rank "+string(rune('0'+rank)), rogue.Eviscerate.DamageMultiplier, want)
	}
}

// Lethality (14128): curve 83046, 10/20% over two ranks.
func TestLethalityIsTenPercentPerRank(t *testing.T) {
	rogue := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"lethality": 2}), "combat_backstab_prebis").GetRogue()
	// core.Spell stores the crit multiplier as 1 + the configured bonus.
	assertClose(t, "Backstab.CritDamageBonus", rogue.Backstab.CritDamageBonus, 1.20)
	assertClose(t, "SinisterStrike.CritDamageBonus", rogue.SinisterStrike.CritDamageBonus, 1.20)
}

// Opportunity (14057): five ranks in 70755, curves 83021/110341 at 4% a
// rank (Backstab, Garrote, Ambush, Mutilate).
func TestOpportunityIsFourPercentPerRank(t *testing.T) {
	rogue := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"opportunity": 5}), "combat_backstab_prebis").GetRogue()
	assertClose(t, "Backstab.DamageMultiplier", rogue.Backstab.DamageMultiplier, 1.5*1.20)
	assertClose(t, "Ambush.DamageMultiplier", rogue.Ambush.DamageMultiplier, 2.5*1.20)
	assertClose(t, "Garrote.DamageMultiplier", rogue.Garrote.DamageMultiplier, 1.20)
}

// Dual Wield Specialization (13715, aura 122): two ranks in 70755, curve
// 128136, 25/50%.
func TestDualWieldSpecializationIsTwentyFivePercentPerRank(t *testing.T) {
	plain := buildRogueForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis").GetRogue()
	spec := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"dual_wield_specialization": 2}), "combat_backstab_prebis").GetRogue()
	assertClose(t, "off-hand damage multiplier ratio",
		spec.AutoAttacks.OHConfig().DamageMultiplier/plain.AutoAttacks.OHConfig().DamageMultiplier, 1.50)
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

// Aggression (18427): curve 83028, 5/10/15%; its 70755 mask (131074) is
// Sinister Strike and Eviscerate only, so Backstab takes none. It adds to
// Coup de Grace on Eviscerate (both aura 108).
func TestAggressionReachesSinisterStrikeAndEviscerateOnly(t *testing.T) {
	talents := talentsWithRanks(t, map[string]int{"aggression": 3, "opportunity": 5, "coup_de_grace": 3})
	rogue := buildRogueForTalentTest(t, talents, "combat_backstab_prebis").GetRogue()
	assertClose(t, "SinisterStrike.DamageMultiplier", rogue.SinisterStrike.DamageMultiplier, 1.15)
	assertClose(t, "Backstab.DamageMultiplier", rogue.Backstab.DamageMultiplier, 1.5*1.20)
	assertClose(t, "Eviscerate.DamageMultiplier", rogue.Eviscerate.DamageMultiplier, 1+0.20+0.15)
}

// Serrated Blades: "Causes your attacks to ignore 3%/6%/9% of your target's
// Armor and increases the damage dealt by your Rupture ability by
// 10%/20%/30%." The client's effect is aura 280 (SpellEffect 690915, 9 at
// rank 3): ignore a percentage of the target's armor, the same shape as the
// warrior's Weaponmaster mace clause. It is not flat armor penetration: the
// engine once added 9 x 13.99 "rating" (a TBC rating constant), which the
// armor formula subtracts as 126 flat armor, more than 9% of a fully
// debuffed raid boss and far less than 9% of an undebuffed one.
func TestSerratedBladesIgnoresThreePercentOfArmorPerRank(t *testing.T) {
	plain := buildRogueForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis").GetRogue()
	serrated := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"serrated_blades": 3}), "combat_backstab_prebis").GetRogue()
	assertClose(t, "armor ignored (fraction of the target's armor) at rank 3", serrated.PseudoStats.ArmorIgnorePercent-plain.PseudoStats.ArmorIgnorePercent, 0.09)
	assertClose(t, "flat armor penetration from Serrated Blades", serrated.GetStat(stats.ArmorPenetration)-plain.GetStat(stats.ArmorPenetration), 0)
	assertClose(t, "Rupture.DamageMultiplier", serrated.Rupture.DamageMultiplier, 1.3)
}

// Venom and Vile Poisons: the client gives both the same two aura=108
// (percent spell modifier) effects on the same poison spell-class masks and
// the same misc values (Venom spell 1310703 effects 1340417/1340418, 30
// each; Vile Poisons spell 16513 effects 693770/693771, 4 per rank). Equal
// aura kind, misc value and mask stack additively in the client, so a poison
// with Venom up and Vile Poisons 5/5 deals 1 + 0.30 + 0.20, not 1.30 * 1.20.
func TestVenomAddsToVilePoisonsInsteadOfMultiplying(t *testing.T) {
	talents := talentsWithRanks(t, map[string]int{"vile_poisons": 5, "venom": 1})
	rogue := buildRogueForTalentTest(t, talents, "combat_backstab_prebis").GetRogue()

	assertClose(t, "InstantPoison before Venom", rogue.InstantPoison.DamageMultiplier, 1.20)
	rogue.VenomAura.OnGain(rogue.VenomAura, nil)
	assertClose(t, "InstantPoison with Venom", rogue.InstantPoison.DamageMultiplier, 1+0.30+0.20)
	assertClose(t, "DeadlyPoison tick with Venom", rogue.DeadlyPoisonTickMultiplierForTest(), 1+0.30+0.20)
	// Wound Poison's mask (268435456) is outside both talents' masks.
	assertClose(t, "WoundPoison with Venom", rogue.WoundPoison.DamageMultiplier, 1)
	rogue.VenomAura.OnExpire(rogue.VenomAura, nil)
	assertClose(t, "InstantPoison after Venom", rogue.InstantPoison.DamageMultiplier, 1.20)
}

// Improved Expose Armor: "Reduces the Energy cost of your Expose Armor
// ability by 5/10, and refunds 1/2 Combo Points when cast with 5 Combo
// Points." (the engine kept Classic's +25%/+50% armor reduction instead).
func TestImprovedExposeArmorCutsEnergyCostByFiveAndTenPerRank(t *testing.T) {
	for rank, want := range map[int]float64{0: 25, 1: 20, 2: 15} {
		rogue := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"improved_expose_armor": rank}), "combat_backstab_prebis").GetRogue()
		assertClose(t, "Expose Armor energy cost at rank "+string(rune('0'+rank)), rogue.ExposeArmor.DefaultCast.Cost, want)
	}
}

func TestImprovedExposeArmorRefundsComboPointsOnlyAtFive(t *testing.T) {
	for _, tc := range []struct{ rank, comboPoints, want int32 }{
		{0, 5, 0}, {1, 5, 1}, {2, 5, 2}, {2, 4, 0}, {2, 1, 0},
	} {
		if got := rogue.ImprovedExposeArmorRefundForTest(tc.rank, tc.comboPoints); got != tc.want {
			t.Errorf("rank %d with %d combo points refunds %d, want %d", tc.rank, tc.comboPoints, got, tc.want)
		}
	}
}

// Deflection: "Increases your Parry chance by 2%/4%/6%." (the engine had 1%
// a rank).
func TestDeflectionIsTwoPercentPerRank(t *testing.T) {
	base := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{}), "combat_backstab_prebis").GetRogue()
	deflected := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"deflection": 3}), "combat_backstab_prebis").GetRogue()
	assertClose(t, "Parry gained at 3/3", deflected.GetStat(stats.Parry)-base.GetStat(stats.Parry), 6*core.ParryRatingPerParryChance)
}
