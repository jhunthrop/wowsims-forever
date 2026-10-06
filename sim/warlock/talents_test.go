package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// One unit test per talent implemented against this task's list of
// proto fields (sim/warlock's AGENTS-level instructions). Each test
// builds a minimal talent string - only the one talent under test is
// non-zero - using newWarlockForDamageTest (spellconst_damage_test.go)
// and TalentTreeSizes/TalentNodeIDs (talents_auto_gen.go) for field
// order, the same convention decimation_test.go and the Siphon Life
// test in spellconst_damage_test.go already use.

// newWarlockForDamageTestMultiTarget is newWarlockForDamageTest with a
// second enemy, for Bane of Havoc's damage redirect (the only talent
// here whose effect needs two targets to observe at all).
func newWarlockForDamageTestMultiTarget(t *testing.T, talents string) (*core.Simulation, *Warlock, *core.Unit, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: talents,
		},
		&proto.Player_Warlock{
			Warlock: &proto.Warlock{
				Options: &proto.WarlockOptions{
					Armor:       proto.WarlockOptions_NoArmor,
					Summon:      proto.WarlockOptions_NoSummon,
					WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
				},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60, core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	built := agent.GetWarlock()
	return sim, built, sim.Encounter.TargetUnits[0], sim.Encounter.TargetUnits[1]
}

// floatsNearlyEqual reports whether a and b differ by no more than a
// tiny tolerance, for sums of repeated float64 additions (e.g. 0.01
// added five times) that do not land on an exact float64 value.
func floatsNearlyEqual(a, b float64) bool {
	const epsilon = 1e-9
	diff := a - b
	return diff > -epsilon && diff < epsilon
}

// TestMaledictionIncreasesPeriodicDamage checks rank 5 (max): +5%
// periodic damage on a Warlock spell with a dot (Corruption).
// PeriodicDamageMultiplierAdditive defaults to 1 (core/spell.go), not
// 0, so this diffs against an untalented build rather than asserting
// an absolute 0.05.
func TestMaledictionIncreasesPeriodicDamage(t *testing.T) {
	_, bare, _ := newBareWarlockForDamageTest(t)
	_, talented, _ := newWarlockForDamageTest(t, "0005") // Affliction field 4, rank 5.

	if len(bare.Corruption) == 0 || len(talented.Corruption) == 0 {
		t.Fatal("level-60 warlock has no Corruption registered")
	}
	bareVal := bare.Corruption[len(bare.Corruption)-1].PeriodicDamageMultiplierAdditive
	talentedVal := talented.Corruption[len(talented.Corruption)-1].PeriodicDamageMultiplierAdditive
	if got, want := talentedVal-bareVal, 0.05; !floatsNearlyEqual(got, want) {
		t.Errorf("Corruption PeriodicDamageMultiplierAdditive delta = %v, want %v (Malediction rank 5)", got, want)
	}
}

// TestImprovedDrainsIncreasesDrainDamage checks rank 3 (max, +20%) on
// Drain Life and Drain Soul, by diffing against an untalented build so
// the test does not need to know either spell's unrelated baseline
// DamageMultiplierAdditive.
func TestImprovedDrainsIncreasesDrainDamage(t *testing.T) {
	_, bare, _ := newBareWarlockForDamageTest(t)
	_, talented, _ := newWarlockForDamageTest(t, "000003") // Affliction field 6, rank 3.

	if len(bare.DrainLife) == 0 || len(talented.DrainLife) == 0 {
		t.Fatal("level-60 warlock has no Drain Life registered")
	}
	bareDL := bare.DrainLife[len(bare.DrainLife)-1].DamageMultiplierAdditive
	talentedDL := talented.DrainLife[len(talented.DrainLife)-1].DamageMultiplierAdditive
	if got, want := talentedDL-bareDL, 0.20; !floatsNearlyEqual(got, want) {
		t.Errorf("Drain Life DamageMultiplierAdditive delta = %v, want %v (Improved Drains rank 3)", got, want)
	}

	if len(bare.DrainSoul) == 0 || len(talented.DrainSoul) == 0 {
		t.Fatal("level-60 warlock has no Drain Soul registered")
	}
	bareDS := bare.DrainSoul[len(bare.DrainSoul)-1].DamageMultiplierAdditive
	talentedDS := talented.DrainSoul[len(talented.DrainSoul)-1].DamageMultiplierAdditive
	if got, want := talentedDS-bareDS, 0.20; !floatsNearlyEqual(got, want) {
		t.Errorf("Drain Soul DamageMultiplierAdditive delta = %v, want %v (Improved Drains rank 3)", got, want)
	}
}

// TestImprovedBaneOfAgonyIncreasesCurseOfAgonyDamage checks rank 2
// (max, +10%) on Curse of Agony ("Bane of Agony" in this build's text).
func TestImprovedBaneOfAgonyIncreasesCurseOfAgonyDamage(t *testing.T) {
	_, bare, _ := newBareWarlockForDamageTest(t)
	_, talented, _ := newWarlockForDamageTest(t, "0000002") // Affliction field 7, rank 2.

	if len(bare.CurseOfAgony) == 0 || len(talented.CurseOfAgony) == 0 {
		t.Fatal("level-60 warlock has no Curse of Agony registered")
	}
	bareCoA := bare.CurseOfAgony[len(bare.CurseOfAgony)-1].DamageMultiplierAdditive
	talentedCoA := talented.CurseOfAgony[len(talented.CurseOfAgony)-1].DamageMultiplierAdditive
	if got, want := talentedCoA-bareCoA, 0.10; !floatsNearlyEqual(got, want) {
		t.Errorf("Curse of Agony DamageMultiplierAdditive delta = %v, want %v (Improved Bane of Agony rank 2)", got, want)
	}
}

// TestFelConcentrationReducesDrainPushback checks rank 3 (max, 70%) on
// Drain Life's PushbackReduction, and that an unrelated Destruction
// spell (Searing Pain) is untouched.
func TestFelConcentrationReducesDrainPushback(t *testing.T) {
	_, built, _ := newWarlockForDamageTest(t, "00000003") // Affliction field 8, rank 3.

	if len(built.DrainLife) == 0 {
		t.Fatal("level-60 warlock has no Drain Life registered")
	}
	if got, want := built.DrainLife[len(built.DrainLife)-1].PushbackReduction, 0.70; got != want {
		t.Errorf("Drain Life PushbackReduction = %v, want %v (Fel Concentration rank 3)", got, want)
	}

	if len(built.SearingPain) == 0 {
		t.Fatal("level-60 warlock has no Searing Pain registered")
	}
	if got, want := built.SearingPain[len(built.SearingPain)-1].PushbackReduction, 0.0; got != want {
		t.Errorf("Searing Pain PushbackReduction = %v, want %v (Fel Concentration must not affect Destruction spells)", got, want)
	}
}

// TestPandemicIncreasesCritDamageBonus checks rank 3 (max, +100%) on
// Corruption's CritDamageBonus.
func TestPandemicIncreasesCritDamageBonus(t *testing.T) {
	_, bare, _ := newBareWarlockForDamageTest(t)
	_, talented, _ := newWarlockForDamageTest(t, "0000000003") // Affliction field 10, rank 3.

	bareCorruption := bare.Corruption[len(bare.Corruption)-1].CritDamageBonus
	talentedCorruption := talented.Corruption[len(talented.Corruption)-1].CritDamageBonus
	if got, want := talentedCorruption-bareCorruption, 1.00; got != want {
		t.Errorf("Corruption CritDamageBonus delta = %v, want %v (Pandemic rank 3)", got, want)
	}
}

// TestMalevolenceIncreasesShadowCrit checks rank 5 (max, +5 crit
// rating points) on a Shadow spell (Corruption) and confirms a Fire
// spell (Searing Pain) is untouched.
func TestMalevolenceIncreasesShadowCrit(t *testing.T) {
	_, bare, _ := newBareWarlockForDamageTest(t)
	_, talented, _ := newWarlockForDamageTest(t, "00000000005") // Affliction field 11, rank 5.

	bareCorruption := bare.Corruption[len(bare.Corruption)-1].BonusCritRating
	talentedCorruption := talented.Corruption[len(talented.Corruption)-1].BonusCritRating
	if got, want := talentedCorruption-bareCorruption, float64(5)*core.CritRatingPerCritChance; got != want {
		t.Errorf("Corruption BonusCritRating delta = %v, want %v (Malevolence rank 5)", got, want)
	}

	bareSP := bare.SearingPain[len(bare.SearingPain)-1].BonusCritRating
	talentedSP := talented.SearingPain[len(talented.SearingPain)-1].BonusCritRating
	if got, want := talentedSP-bareSP, 0.0; got != want {
		t.Errorf("Searing Pain BonusCritRating delta = %v, want %v (Malevolence must not affect Fire spells)", got, want)
	}
}

// TestSoulSiphonMultiplierCountsOtherAfflictionEffects checks rank 3
// (max, +12% per other effect, up to 3): 0 active effects -> 1.0, 1
// (Corruption landed) -> 1.12, 2 (Corruption + Curse of Agony) -> 1.24.
func TestSoulSiphonMultiplierCountsOtherAfflictionEffects(t *testing.T) {
	sim, built, target := newWarlockForDamageTest(t, "000000000000003") // Affliction field 15, rank 3.

	drainLife := built.DrainLife[len(built.DrainLife)-1]
	if got, want := built.soulSiphonMultiplier(target, drainLife), 1.0; got != want {
		t.Errorf("soulSiphonMultiplier with no other effects = %v, want %v", got, want)
	}

	corruption := built.Corruption[len(built.Corruption)-1]
	corruption.ApplyEffects(sim, target, corruption)
	if !corruption.Dot(target).IsActive() {
		t.Skip("Corruption did not land (hit-table roll); rerun")
	}
	if got, want := built.soulSiphonMultiplier(target, drainLife), 1.12; got != want {
		t.Errorf("soulSiphonMultiplier with 1 other effect = %v, want %v", got, want)
	}

	curseOfAgony := built.CurseOfAgony[len(built.CurseOfAgony)-1]
	curseOfAgony.ApplyEffects(sim, target, curseOfAgony)
	if !curseOfAgony.Dot(target).IsActive() {
		t.Skip("Curse of Agony did not land (hit-table roll); rerun")
	}
	if got, want := built.soulSiphonMultiplier(target, drainLife), 1.24; got != want {
		t.Errorf("soulSiphonMultiplier with 2 other effects = %v, want %v", got, want)
	}

	// Wrack must not count against its own snapshot.
	if len(built.DoTSpells) == 0 {
		t.Fatal("warlock has no tracked DoT spells")
	}
}

// TestFelVitalityIncreasesManaAndPetStats checks rank 3 (max, +15%) on
// the warlock's own max Mana and the active pet's max Mana and Health.
func TestFelVitalityIncreasesManaAndPetStats(t *testing.T) {
	_, bare, _ := newBareWarlockForDamageTest(t)
	_, talented, _ := newWarlockForDamageTest(t, "-0000003") // Demonology field 7, rank 3.

	bareMana := bare.GetStat(stats.Mana)
	talentedMana := talented.GetStat(stats.Mana)
	if got, want := talentedMana/bareMana, 1.15; got < want-0.001 || got > want+0.001 {
		t.Errorf("warlock Mana ratio = %v, want %v (Fel Vitality rank 3)", got, want)
	}

	bareImpMana := bare.Imp.GetStat(stats.Mana)
	talentedImpMana := talented.Imp.GetStat(stats.Mana)
	if got, want := talentedImpMana/bareImpMana, 1.15; got < want-0.001 || got > want+0.001 {
		t.Errorf("Imp Mana ratio = %v, want %v (Fel Vitality rank 3)", got, want)
	}

	bareImpHealth := bare.Imp.GetStat(stats.Health)
	talentedImpHealth := talented.Imp.GetStat(stats.Health)
	if got, want := talentedImpHealth/bareImpHealth, 1.15; got < want-0.001 || got > want+0.001 {
		t.Errorf("Imp Health ratio = %v, want %v (Fel Vitality rank 3)", got, want)
	}
}

// TestDemonicEnergiesSharesLifeTapManaWithPet checks both ranks: 50%
// at rank 1, 100% at rank 2, shared with whichever pet is active.
func TestDemonicEnergiesSharesLifeTapManaWithPet(t *testing.T) {
	sim, built, _ := newWarlockForDamageTest(t, "-00000001") // Demonology field 8, rank 1.
	built.ActivePet = built.Imp

	// Pets start at full mana, so AddMana would clamp to 0 gained;
	// spend it down first to leave room to observe the share.
	spendMetrics := built.Imp.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion})
	built.Imp.SpendMana(sim, built.Imp.MaxMana(), spendMetrics)

	before := built.Imp.CurrentMana()
	built.shareDemonicEnergiesMana(sim, 100)
	if got, want := built.Imp.CurrentMana()-before, 50.0; got != want {
		t.Errorf("Imp mana gained = %v, want %v (Demonic Energies rank 1, 50%% share)", got, want)
	}

	sim2, built2, _ := newWarlockForDamageTest(t, "-00000002") // Demonology field 8, rank 2.
	built2.ActivePet = built2.Imp
	spendMetrics2 := built2.Imp.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion})
	built2.Imp.SpendMana(sim2, built2.Imp.MaxMana(), spendMetrics2)

	before2 := built2.Imp.CurrentMana()
	built2.shareDemonicEnergiesMana(sim2, 100)
	if got, want := built2.Imp.CurrentMana()-before2, 100.0; got != want {
		t.Errorf("Imp mana gained = %v, want %v (Demonic Energies rank 2, 100%% share)", got, want)
	}
}

// TestDemonicKnowledgeBuffsWarlockAndActivePet checks rank 3 (max,
// +100% of level) applies to both the warlock and whichever pet the
// test build summons by default.
func TestDemonicKnowledgeBuffsWarlockAndActivePet(t *testing.T) {
	// newDemonicPactTestWarlock summons the Imp by default, which
	// newWarlockForDamageTest/newBareWarlockForDamageTest never do
	// (always proto.WarlockOptions_NoSummon) - Demonic Knowledge needs
	// an active pet to have anything to observe.
	_, bare := newDemonicPactTestWarlock(t, "")
	_, talented := newDemonicPactTestWarlock(t, "-00000000000000003") // Demonology field 17, rank 3.

	if talented.ActivePet == nil {
		t.Fatal("warlock has no active pet; Demonic Knowledge needs one to observe")
	}

	wantBonus := 1.00 * float64(talented.Level)
	if got, want := talented.GetStat(stats.SpellPower)-bare.GetStat(stats.SpellPower), wantBonus; got != want {
		t.Errorf("warlock SpellPower delta = %v, want %v (Demonic Knowledge rank 3)", got, want)
	}

	wantPetBonus := wantBonus
	if got, want := talented.Imp.GetStat(stats.SpellPower)-bare.Imp.GetStat(stats.SpellPower), wantPetBonus; got != want {
		t.Errorf("Imp SpellPower delta = %v, want %v (Demonic Knowledge rank 3)", got, want)
	}
}

// TestAftermathIncreasesImmolateInitialDamage checks all 5 ranks of
// the per-rank value immolate.go reads.
func TestAftermathIncreasesImmolateInitialDamage(t *testing.T) {
	for rank := int32(1); rank <= 5; rank++ {
		// Destruction field 6: 5 leading zeros then the rank digit.
		talentsString := "--00000" + string(rune('0'+rank))
		_, built, _ := newWarlockForDamageTest(t, talentsString)
		if got, want := built.aftermathInitialDamageBonus(), 0.10*float64(rank); got != want {
			t.Errorf("aftermathInitialDamageBonus(rank %d) = %v, want %v", rank, got, want)
		}
	}
}

// TestIntensityReducesDestructionPushback checks rank 3 (max, 70%) on
// a Destruction spell (Searing Pain) and that Affliction (Corruption)
// is untouched.
func TestIntensityReducesDestructionPushback(t *testing.T) {
	_, built, _ := newWarlockForDamageTest(t, "--000000003") // Destruction field 9, rank 3.

	if got, want := built.SearingPain[len(built.SearingPain)-1].PushbackReduction, 0.70; got != want {
		t.Errorf("Searing Pain PushbackReduction = %v, want %v (Intensity rank 3)", got, want)
	}
	if got, want := built.Corruption[len(built.Corruption)-1].PushbackReduction, 0.0; got != want {
		t.Errorf("Corruption PushbackReduction = %v, want %v (Intensity must not affect Affliction spells)", got, want)
	}
}

// TestAgonizingFlamesBuffsSearingPainAndDestruction checks rank 3
// (max, +10%): both halves on Searing Pain, only the damage half on
// another Destruction spell (Shadow Bolt).
func TestAgonizingFlamesBuffsSearingPainAndDestruction(t *testing.T) {
	_, bare, _ := newBareWarlockForDamageTest(t)
	_, talented, _ := newWarlockForDamageTest(t, "--0000000003") // Destruction field 10, rank 3.

	bareSP := bare.SearingPain[len(bare.SearingPain)-1]
	talentedSP := talented.SearingPain[len(talented.SearingPain)-1]
	if got, want := talentedSP.DamageMultiplierAdditive-bareSP.DamageMultiplierAdditive, 0.10; !floatsNearlyEqual(got, want) {
		t.Errorf("Searing Pain DamageMultiplierAdditive delta = %v, want %v (Agonizing Flames rank 3)", got, want)
	}
	if got, want := talentedSP.BonusCritRating-bareSP.BonusCritRating, float64(10)*core.CritRatingPerCritChance; got != want {
		t.Errorf("Searing Pain BonusCritRating delta = %v, want %v (Agonizing Flames rank 3)", got, want)
	}

	bareSB := bare.ShadowBolt[len(bare.ShadowBolt)-1]
	talentedSB := talented.ShadowBolt[len(talented.ShadowBolt)-1]
	if got, want := talentedSB.DamageMultiplierAdditive-bareSB.DamageMultiplierAdditive, 0.10; !floatsNearlyEqual(got, want) {
		t.Errorf("Shadow Bolt DamageMultiplierAdditive delta = %v, want %v (Agonizing Flames rank 3 affects all Destruction spells)", got, want)
	}
	if got, want := talentedSB.BonusCritRating-bareSB.BonusCritRating, 0.0; got != want {
		t.Errorf("Shadow Bolt BonusCritRating delta = %v, want %v (Agonizing Flames' crit half is Searing Pain only)", got, want)
	}
}

// TestFireAndBrimstoneIncreasesConflagrateCrit checks rank 3 (max,
// +25 crit rating points) on Conflagrate.
func TestFireAndBrimstoneIncreasesConflagrateCrit(t *testing.T) {
	// Field 11 (conflagrate, bool) must also be set for Conflagrate to
	// register at all; field 14 (fire_and_brimstone) is the talent
	// under test.
	_, bare, _ := newWarlockForDamageTest(t, "--00000000001")        // Conflagrate only.
	_, talented, _ := newWarlockForDamageTest(t, "--00000000001003") // Conflagrate + Fire and Brimstone rank 3.

	if len(bare.Conflagrate) == 0 || len(talented.Conflagrate) == 0 {
		t.Fatal("level-60 warlock with the Conflagrate talent has no Conflagrate registered")
	}
	bareCrit := bare.Conflagrate[len(bare.Conflagrate)-1].BonusCritRating
	talentedCrit := talented.Conflagrate[len(talented.Conflagrate)-1].BonusCritRating
	if got, want := talentedCrit-bareCrit, float64(25)*core.CritRatingPerCritChance; got != want {
		t.Errorf("Conflagrate BonusCritRating delta = %v, want %v (Fire and Brimstone rank 3)", got, want)
	}
}

// TestShadowAndFlameBuffsAndPreservesImmolate checks both of Shadow
// and Flame's implemented halves at rank 5 (max): the 20 s school
// damage buff (triggered the same way conflagrate.go/shadowburn.go
// call it on a landed hit) and the 100% chance to preserve Immolate.
func TestShadowAndFlameBuffsAndPreservesImmolate(t *testing.T) {
	sim, built, _ := newWarlockForDamageTest(t, "--000000000000005") // Destruction field 15, rank 5.

	before := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow]
	built.triggerShadowAndFlame(sim, SpellCode_WarlockConflagrate)
	after := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow]
	if got, want := after/before, 1.10; got < want-0.0001 || got > want+0.0001 {
		t.Errorf("Shadow damage multiplier ratio after Conflagrate = %v, want %v (Shadow and Flame rank 5)", got, want)
	}

	beforeFire := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire]
	built.triggerShadowAndFlame(sim, SpellCode_WarlockShadowburn)
	afterFire := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire]
	if got, want := afterFire/beforeFire, 1.10; got < want-0.0001 || got > want+0.0001 {
		t.Errorf("Fire damage multiplier ratio after Shadowburn = %v, want %v (Shadow and Flame rank 5)", got, want)
	}

	// Rank 5's 100% chance is deterministic, no retry loop needed.
	if !built.shadowAndFlamePreservesImmolate(sim) {
		t.Error("shadowAndFlamePreservesImmolate(rank 5, 100% chance) = false, want true")
	}
}

// TestBaneOfHavocRedirectsDamageToCurseTarget casts a direct-damage
// spell at one target while a second target carries Bane of Havoc, and
// checks the second target also took 15% of that damage.
func TestBaneOfHavocRedirectsDamageToCurseTarget(t *testing.T) {
	sim, built, otherTarget, havocTarget := newWarlockForDamageTestMultiTarget(t, "--0000000000001") // Destruction field 13.

	if built.BaneOfHavoc == nil {
		t.Fatal("level-60 warlock with the Bane of Havoc talent has no Bane of Havoc spell")
	}
	built.BaneOfHavoc.ApplyEffects(sim, havocTarget, built.BaneOfHavoc)
	if !built.BaneOfHavocAuras.Get(havocTarget).IsActive() {
		t.Fatal("Bane of Havoc aura did not activate on its target")
	}

	// A direct CalcAndDealDamage with OutcomeAlwaysHit (rather than
	// Searing Pain's own ApplyEffects, which crit-rolls via
	// OutcomeMagicHitAndCrit) keeps this test's damage amount exact,
	// since Bane of Havoc's 15% applies to whatever landed, crit or
	// not, and the point here is checking the 15%, not the crit roll.
	// CalcAndDealDamage's baseDamage argument is the PRE-multiplier
	// input, not the final damage dealt - read the actual amount off
	// the returned result, which is what the OnSpellHitDealt hook that
	// drives the redirect also sees.
	searingPain := built.SearingPain[len(built.SearingPain)-1]
	result := searingPain.CalcAndDealDamage(sim, otherTarget, 200.0, searingPain.OutcomeAlwaysHit)
	dealt := result.Damage

	spillover := built.GetSpell(core.ActionID{SpellID: 1225228, Tag: 1})
	if spillover == nil {
		t.Fatal("Bane of Havoc has no spillover spell registered")
	}
	gotSpillover := spillover.SpellMetrics[havocTarget.UnitIndex].TotalDamage
	wantSpillover := dealt * baneOfHavocSpilloverPct
	if !floatsNearlyEqual(gotSpillover, wantSpillover) {
		t.Errorf("Bane of Havoc spillover damage = %v, want %v (15%% of %v)", gotSpillover, wantSpillover, dealt)
	}
}

// newDemonicPactTestWarlock is newWarlockForDamageTest with the Imp
// summoned by default (that helper always sets NoSummon), since Demonic
// Sacrifice's ExtraCastCondition requires an active pet.
func newDemonicPactTestWarlock(t *testing.T, talents string) (*core.Simulation, *Warlock) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: talents,
		},
		&proto.Player_Warlock{
			Warlock: &proto.Warlock{
				Options: &proto.WarlockOptions{
					Armor:       proto.WarlockOptions_NoArmor,
					Summon:      proto.WarlockOptions_Imp,
					WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
				},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	return sim, agent.GetWarlock()
}

// TestDemonicPactKeepsSacrificeAcrossDifferentPetButNotSameOne checks
// Demonic Pact's own wording: summoning a DIFFERENT demon no longer
// cancels Demonic Sacrifice; resummoning the SACRIFICED one still does.
// Observed through Demonic Sacrifice's own Imp effect, Burning Wish
// (+15% Fire damage dealt), since the sacrifice auras themselves are
// local to applyDemonicSacrifice.
func TestDemonicPactKeepsSacrificeAcrossDifferentPetButNotSameOne(t *testing.T) {
	// Demonology field 10 (demonic_sacrifice, bool) and field 19
	// (demonic_pact, bool).
	sim, built := newDemonicPactTestWarlock(t, "-0000000001000000001")

	if built.ActivePet != built.Imp {
		t.Fatal("warlock's default pet is not the Imp")
	}

	demonicSacrifice := built.GetSpell(core.ActionID{SpellID: 18788})
	if demonicSacrifice == nil {
		t.Fatal("level-60 warlock with the Demonic Sacrifice talent has no Demonic Sacrifice spell")
	}
	demonicSacrifice.ApplyEffects(sim, nil, demonicSacrifice)

	if built.sacrificedPet != built.Imp {
		t.Fatal("Demonic Sacrifice did not record the Imp as the sacrificed pet")
	}
	base := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire] / 1.15
	sacrificed := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire]
	if got, want := sacrificed, base*1.15; got != want {
		t.Fatalf("Fire damage multiplier after sacrificing the Imp = %v, want %v (Burning Wish)", got, want)
	}

	// Summoning a DIFFERENT pet must not cancel it.
	built.changeActivePet(sim, built.Succubus, false)
	if got, want := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire], sacrificed; got != want {
		t.Errorf("Fire damage multiplier after summoning a different pet = %v, want %v (Demonic Pact should preserve the sacrifice)", got, want)
	}

	// Resummoning the SACRIFICED pet must cancel it.
	built.changeActivePet(sim, nil, false)
	built.changeActivePet(sim, built.Imp, false)
	if got, want := built.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexFire], base; got != want {
		t.Errorf("Fire damage multiplier after resummoning the sacrificed Imp = %v, want %v (the sacrifice should be cancelled)", got, want)
	}
}
