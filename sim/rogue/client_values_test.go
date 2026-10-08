package rogue_test

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/rogue"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
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

// Aggression: "Increases the damage of your Sinister Strike, Backstab, and
// Eviscerate abilities by 2%/4%/6%." Backstab had no Aggression term; it adds
// to Opportunity the way it adds to Improved Eviscerate on Eviscerate.
func TestAggressionReachesSinisterStrikeBackstabAndEviscerate(t *testing.T) {
	talents := talentsWithRanks(t, map[string]int{"aggression": 3, "opportunity": 2, "improved_eviscerate": 3})
	rogue := buildRogueForTalentTest(t, talents, "combat_backstab_prebis").GetRogue()
	assertClose(t, "SinisterStrike.DamageMultiplier", rogue.SinisterStrike.DamageMultiplier, 1.06)
	assertClose(t, "Backstab.DamageMultiplier", rogue.Backstab.DamageMultiplier, 1.5*(1+0.10+0.06))
	assertClose(t, "Eviscerate.DamageMultiplier", rogue.Eviscerate.DamageMultiplier, 1+0.20+0.06)
}

// Murder: "Increases all damage dealt by 2%/4% against Humanoid and Giant
// targets." The engine had 1%/2%, applied to Beast and Dragonkin too, and
// multiplied the crit multiplier as well, which counted it twice on a crit.
func TestMurderIsTwoAndFourPercentAgainstHumanoidAndGiantOnly(t *testing.T) {
	cases := []struct {
		mobType proto.MobType
		want    float64
	}{
		{proto.MobType_MobTypeHumanoid, 1.04},
		{proto.MobType_MobTypeGiant, 1.04},
		{proto.MobType_MobTypeBeast, 1},
		{proto.MobType_MobTypeDragonkin, 1},
		{proto.MobType_MobTypeUnknown, 1},
	}
	for _, c := range cases {
		built := buildRogueAgainst(t, talentsWithRanks(t, map[string]int{"murder": 2}), c.mobType)
		table := built.GetRogue().AttackTables[0][proto.CastType_CastTypeMainHand]
		assertClose(t, c.mobType.String()+" damage multiplier", table.DamageDealtMultiplier, c.want)
		assertClose(t, c.mobType.String()+" crit multiplier", table.CritMultiplier, 1)
	}
}

func buildRogueAgainst(t *testing.T, talentsStr string, mobType proto.MobType) *dpsrogue.DpsRogue {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassRogue,
			Race:               proto.Race_RaceHuman,
			Equipment:          core.GetGearSet("../../ui/rogue/gear_sets", "combat_backstab_prebis").GearSet,
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
		},
		&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{Targets: []*proto.Target{{Level: 63, MobType: mobType}}}
	env, _, _ := core.NewEnvironment(raid, encounter, true)
	built, ok := env.Raid.Parties[0].Players[0].(*dpsrogue.DpsRogue)
	if !ok {
		t.Fatal("player 0 did not build as a *dpsrogue.DpsRogue")
	}
	return built
}

// Serrated Blades: "Causes your attacks to ignore 3%/6%/9% of your target's
// Armor and increases the damage dealt by your Rupture ability by
// 10%/20%/30%." The engine's integer-division term (5/3 is 1) gave one point
// of armor penetration rating per level, about 4.3% per rank.
func TestSerratedBladesIgnoresThreePercentOfArmorPerRank(t *testing.T) {
	plain := buildRogueForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis").GetRogue()
	serrated := buildRogueForTalentTest(t, talentsWithRanks(t, map[string]int{"serrated_blades": 3}), "combat_backstab_prebis").GetRogue()
	got := (serrated.GetStat(stats.ArmorPenetration) - plain.GetStat(stats.ArmorPenetration)) / core.ArmorPenPerPercentArmor
	assertClose(t, "armor ignored (percent) at rank 3", got, 9)
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
	assertClose(t, "WoundPoison with Venom", rogue.WoundPoison.DamageMultiplier, 1+0.30+0.20)
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
