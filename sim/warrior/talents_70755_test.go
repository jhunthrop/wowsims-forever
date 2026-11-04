package warrior

import (
	"testing"
	"time"
)

// The tests in this file pin the warrior talents and spells the
// 1.60.1.70755 hotfix push added or retuned to the client's own rows: the
// conformance client table (sim/core/testdata/conformance/client/
// warrior.json, generated from the layered 70755 tables) for each spell's
// max-rank row, and the per-rank trait curves (TraitDefinitionEffectPoints
// -> CurvePoint) quoted at each table.

// clientAuraAddFlatModifier is SPELL_AURA_ADD_FLAT_MODIFIER; its misc
// value names the modified field (10 cast time, 11 cooldown, 14 cost,
// 21 global cooldown).
const clientAuraAddFlatModifier = 107

// Improved Whirlwind rank 2 cuts Whirlwind's cooldown by 2 s and raises
// its damage to the closest enemy by 30%; rank 1 by 1 s and 15%.
func TestImprovedWhirlwindRanks(t *testing.T) {
	cases := []struct {
		points     int32
		cut        time.Duration
		multiplier float64
	}{
		{0, 0, 1},
		{1, time.Second, 1.15},
		{2, 2 * time.Second, 1.30},
		{9, 2 * time.Second, 1.30},
	}
	for _, c := range cases {
		if got := improvedWhirlwindCooldownCut(c.points); got != c.cut {
			t.Errorf("Improved Whirlwind %d: cooldown cut %s, want %s", c.points, got, c.cut)
		}
		if got := improvedWhirlwindPrimaryTargetMultiplier(c.points); got != c.multiplier {
			t.Errorf("Improved Whirlwind %d: closest-enemy multiplier %v, want %v", c.points, got, c.multiplier)
		}
	}
}

// The max rank is the client row 1310315: e0 aura 107 misc 11 -2000 on
// Whirlwind's family bit (mask_1 0x4, which 1680 carries), e1 aura 4 30.
func TestImprovedWhirlwindMaxRankIsTheClientRow(t *testing.T) {
	client := clientWarrior(t)
	row, ok := client.ByID(TalentSpellIDs["improved_whirlwind"][0])
	if !ok || len(row.Effects) != 2 {
		t.Fatalf("Improved Whirlwind 1310315 is not in the client table with two effects")
	}
	maxRank := int32(len(TalentSpellIDs["improved_whirlwind"]))
	cd, bonus := row.Effects[0], row.Effects[1]
	if cd.Aura != clientAuraAddFlatModifier || cd.MiscValue != 11 {
		t.Errorf("1310315 e0 is aura %d misc %d, want the cooldown modifier 107/11", cd.Aura, cd.MiscValue)
	}
	if got, want := improvedWhirlwindCooldownCut(maxRank), -time.Duration(cd.Amount)*time.Millisecond; got != want {
		t.Errorf("Improved Whirlwind %d/%d cuts %s, client e0 %s", maxRank, maxRank, got, want)
	}
	if got, want := improvedWhirlwindPrimaryTargetMultiplier(maxRank), 1+bonus.Amount/100; got != want {
		t.Errorf("Improved Whirlwind %d/%d multiplies by %v, client e1 %v", maxRank, maxRank, got, want)
	}
	whirlwind, ok := client.ByID(1680)
	if !ok {
		t.Fatal("Whirlwind 1680 is not in the client table")
	}
	if whirlwind.FamilyMask[1]&4 == 0 {
		t.Errorf("Whirlwind's family mask %v lacks the mask_1 0x4 bit Improved Whirlwind modifies", whirlwind.FamilyMask)
	}
	if got := time.Duration(whirlwind.CategoryCooldownMS) * time.Millisecond; got != whirlwindCooldown {
		t.Errorf("Whirlwind's cooldown is %s, client %s", whirlwindCooldown, got)
	}
}

// Flurry is 10%/20%/30% melee attack speed at ranks 1-3 since
// 1.60.1.70755 (trait curve 83289), and the Protection T2 4pc's Flurry is
// the max rank's 30%.
func TestFlurryIsTenPercentAPointOverThreeRanks(t *testing.T) {
	if got := flurryMaxPoints(); got != 3 {
		t.Fatalf("Flurry has %d client ranks, want 3", got)
	}
	for points, want := range map[int32]float64{1: 1.10, 2: 1.20, 3: 1.30, 5: 1.30} {
		if got := flurryAttackSpeed[rankIndex(points, flurryAttackSpeed[:])]; got != want {
			t.Errorf("Flurry %d: attack speed %v, want %v", points, got, want)
		}
	}
}

// Two-Handed Weapon Specialization is 2%/4%/6% (trait curve 83308).
func TestTwoHandedWeaponSpecializationIsTwoPercentAPoint(t *testing.T) {
	for points, want := range map[int32]float64{0: 1, 1: 1.02, 2: 1.04, 3: 1.06, 9: 1.06} {
		if got := twoHandedWeaponSpecializationMultiplier(points); got != want {
			t.Errorf("Two-Handed Weapon Specialization %d: multiplier %v, want %v", points, got, want)
		}
	}
	if got := len(TalentSpellIDs["two_handed_weapon_specialization"]); got != 3 {
		t.Errorf("Two-Handed Weapon Specialization has %d ranks, want 3", got)
	}
}

// Booming Voice's cost clause (12321 e1, aura 108 misc 14, -25 at 5/5)
// reaches Piercing Howl since 1.60.1.70755: the e1 mask carries 12323's
// family bit (mask_1 0x20) beside the shouts'.
func TestBoomingVoiceReachesHowls(t *testing.T) {
	client := clientWarrior(t)
	booming, ok := client.ByID(TalentSpellIDs["booming_voice"][0])
	if !ok || len(booming.Effects) < 2 {
		t.Fatal("Booming Voice 12321 is not in the client table with two effects")
	}
	howl, ok := client.ByID(PiercingHowlSpellId[0])
	if !ok {
		t.Fatal("Piercing Howl 12323 is not in the client table")
	}
	if howl.FamilyMask[1] != 32 {
		t.Errorf("Piercing Howl's family mask %v, want mask_1 0x20", howl.FamilyMask)
	}
	if got, want := -float64(boomingVoiceCostReductionPct*len(TalentSpellIDs["booming_voice"])), booming.Effects[1].Amount; got != want {
		t.Errorf("Booming Voice 5/5 takes %v%% off, client e1 %v", got, want)
	}
	if boomingVoiceClassMask&WarriorSpellMaskPiercingHowl == 0 {
		t.Error("Booming Voice's cost mod does not reach Piercing Howl")
	}
}

// Reckless Swings' rows: 15 s cooldown, a 1 s global cooldown, 8 s
// duration, a health cost of 25% (power type -2), and a proc trigger of
// 1329554, whose energize (effect 30, misc 1 rage) is 50 tenths: 5 rage.
func TestRecklessSwingsMatchesTheClient(t *testing.T) {
	client := clientWarrior(t)
	cast, ok := client.ByID(RecklessSwingsSpellId[0])
	if !ok {
		t.Fatal("Reckless Swings 1329555 is not in the client table")
	}
	if got, want := RecklessSwingsCooldownMS[0], cast.CooldownMS; got != want || want != 15000 {
		t.Errorf("Reckless Swings cooldown %d ms, client %d (want 15000)", got, want)
	}
	if got := time.Duration(cast.GCDMS) * time.Millisecond; got != recklessSwingsGCD {
		t.Errorf("Reckless Swings GCD %s, client %s", recklessSwingsGCD, got)
	}
	if got := time.Duration(cast.DurationMS) * time.Millisecond; got != recklessSwingsDuration {
		t.Errorf("Reckless Swings lasts %s, client %s", recklessSwingsDuration, got)
	}
	const clientPowerHealth = -2
	if cast.CostType != clientPowerHealth || cast.CostPct != RecklessSwingsManaCostPct[0] || cast.CostPct != 25 {
		t.Errorf("Reckless Swings costs type %d, %v%%; want health (-2), 25%% (engine reads %v%%)", cast.CostType, cast.CostPct, RecklessSwingsManaCostPct[0])
	}
	if len(cast.Effects) != 1 || cast.Effects[0].Aura != 42 || cast.Effects[0].TriggerSpell != recklessSwingsRageSpellID {
		t.Fatalf("1329555's effects %+v; want one aura 42 triggering %d", cast.Effects, recklessSwingsRageSpellID)
	}
	rage, ok := client.ByID(recklessSwingsRageSpellID)
	if !ok {
		t.Fatalf("Reckless Swings' rage spell %d is not in the client table", recklessSwingsRageSpellID)
	}
	energize := rage.Effects[0]
	if energize.Effect != 30 || energize.MiscValue != 1 || energize.Amount/10 != recklessSwingsRage {
		t.Errorf("1329554 e0 is effect %d misc %d amount %v; want an energize of rage worth %v", energize.Effect, energize.MiscValue, energize.Amount, recklessSwingsRage)
	}
	// SpellAuraOptions 249144 (not in the conformance table): ProcCharges 4.
	if recklessSwingsCharges != 4 {
		t.Errorf("Reckless Swings has %d charges, want the client's 4", recklessSwingsCharges)
	}
}
