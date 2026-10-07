package hunter

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// Every proto.HunterTalents field this file's ApplyTalents actually
// reads, named by its talents_auto_gen.go proto field name. Keep this
// in step with talents.go the way sim/mage/talents_test.go's own
// foreverFrostTalentsApplied list is kept in step with its talents.go.
var hunterTalentsApplied = []string{
	// Beast Mastery
	"deadly_aspects",
	"focused_fire",
	// Marksmanship
	"lethal_attacks",
	"improved_stings",
	"careful_aim",
	"rapid_killing",
	"lone_wolf",
	"trueshot_aura",
	"rapid_recuperation",
	"sniper_shot",
	// Beast Mastery (new spell, registered from hunter.go's Initialize)
	"summon_hawk",
	// Survival
	"improved_tracking",
	"resourcefulness",
	"expose_prey",
	"survivalists_discipline",
}

func TestEveryTalentThisPackageAppliesExists(t *testing.T) {
	for _, name := range hunterTalentsApplied {
		if _, ok := TalentNodeIDs[name]; !ok {
			t.Errorf("talents.go applies %q, which is not in the client's tree", name)
		}
		if len(TalentSpellIDs[name]) == 0 {
			t.Errorf("%q has no rank spell ids", name)
		}
	}
}

// buildTalentsString spends `rank` points in `field` and nothing else,
// by generated proto field name (core.TalentsStringFromRanks), against
// the same positional contract core.FillTalentsProto reads (no
// prerequisite validation, confirmed by sim/mage/talents.go's own rankOf
// comment), so a single-talent build needs no other point spent to be
// legal and a tree layout change cannot retarget it.
func buildTalentsString(t *testing.T, field string, rank int32) string {
	t.Helper()
	str, err := core.TalentsStringFromRanks((&proto.HunterTalents{}).ProtoReflect(), TalentTreeSizes, map[string]int{field: int(rank)})
	if err != nil {
		t.Fatal(err)
	}
	return str
}

// buildHunterForTalentTest builds a bare (no gear, no buffs) hunter with
// exactly one talent spent, through core.NewEnvironment directly (no
// core.NewSim / PrePull) - static ApplyTalents + Initialize + finalize
// effects only, which is all the non-proc assertions below need. Mirrors
// sim/hunter/pet_level_test.go's own pattern, which the same comment
// there explains: this fork's item database is not generated in this
// test environment, so Equipment must stay empty.
func buildHunterForTalentTest(t *testing.T, level int32, field string, rank int32, petType proto.Hunter_Options_PetType) *Hunter {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              level,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              &proto.IndividualBuffs{},
			TalentsString:      buildTalentsString(t, field, rank),
			DistanceFromTarget: 5,
		},
		&proto.Player_Hunter{
			Hunter: &proto.Hunter{
				Options: &proto.Hunter_Options{
					Ammo:           proto.Hunter_Options_RazorArrow,
					PetType:        petType,
					PetUptime:      1,
					PetAttackSpeed: 2.0,
				},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})

	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{
		Targets: []*proto.Target{core.DefaultTargetProtoLvl60},
	}, true)

	built, ok := env.Raid.Parties[0].Players[0].(*Hunter)
	if !ok {
		t.Fatal("player 0 did not build as a *Hunter")
	}
	return built
}

// newRunningHunterForTalentTest is buildHunterForTalentTest's shape but
// through a full core.NewSim + Reset + PrePull (newBareHunterAtLevel's
// own shape in spellconst_hunter_test.go), which the proc-based talent
// tests below need for a live *core.Simulation to call sim.Proc and
// Aura.Activate against.
func newRunningHunterForTalentTest(t *testing.T, level int32, field string, rank int32, petType proto.Hunter_Options_PetType, distanceFromTarget float64, debuffs *proto.Debuffs) (*core.Simulation, *Hunter, *core.Unit) {
	t.Helper()

	if debuffs == nil {
		debuffs = &proto.Debuffs{}
	}

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              level,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              &proto.IndividualBuffs{},
			TalentsString:      buildTalentsString(t, field, rank),
			DistanceFromTarget: distanceFromTarget,
		},
		&proto.Player_Hunter{
			Hunter: &proto.Hunter{
				Options: &proto.Hunter_Options{
					Ammo:           proto.Hunter_Options_RazorArrow,
					PetType:        petType,
					PetUptime:      1,
					PetAttackSpeed: 2.0,
				},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, debuffs)

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

	agent, ok := sim.Raid.Parties[0].Players[0].(HunterAgent)
	if !ok {
		t.Fatal("the raid's first player is not a hunter agent")
	}
	built := agent.GetHunter()
	target := sim.Encounter.TargetUnits[0]
	return sim, built, target
}

// TestDeadlyAspectsProcsRangedHasteFromAutoShot is Deadly Aspects at max
// rank (5, a 10% chance per landed Auto Shot): forcing Aspect of the
// Hawk active and replaying a landed ranged-auto hit through the
// registered aura's own OnSpellHitDealt handler 300 times gives a
// (1-0.10)^300 ~= 1.9e-14 chance of a false failure, which is the same
// repeat-and-assert-at-least-once shape a probabilistic proc needs
// without pinning the engine's RNG stream.
func TestDeadlyAspectsProcsRangedHasteFromAutoShot(t *testing.T) {
	sim, hunter, target := newRunningHunterForTalentTest(t, 60, "deadly_aspects", 5, proto.Hunter_Options_PetNone, 25, nil)

	hawkRank := hunter.getMaxHawkRank()
	hawkAura := hunter.GetAura("Aspect of the Hawk" + itoa(hawkRank))
	if hawkAura == nil {
		t.Fatal("level-60 hunter has no Aspect of the Hawk aura registered")
	}
	hawkAura.Activate(sim)
	if !hunter.HasActiveAuraWithTag(AspectOfTheHawkAuraTag) {
		t.Fatal("Aspect of the Hawk aura did not carry AspectOfTheHawkAuraTag active")
	}

	deadlyAspects := hunter.GetAura("Deadly Aspects")
	if deadlyAspects == nil {
		t.Fatal("DeadlyAspects=5 did not register a \"Deadly Aspects\" listener aura")
	}

	landedAutoShot := &core.SpellResult{Outcome: core.OutcomeLanded, Target: target}
	autoShotSpell := &core.Spell{ProcMask: core.ProcMaskRangedAuto}

	proced := false
	for i := 0; i < 300 && !proced; i++ {
		deadlyAspects.OnSpellHitDealt(deadlyAspects, sim, autoShotSpell, landedAutoShot)
		proced = hunter.HasActiveAura("Deadly Aspects Haste")
	}
	if !proced {
		t.Error("Deadly Aspects at max rank never proced its ranged haste buff over 300 landed Auto Shots")
	}
}

// itoa avoids importing strconv for one call site.
func itoa(n int) string {
	return [...]string{"0", "1", "2", "3", "4", "5", "6", "7"}[n]
}

func TestFocusedFireBuffsHunterAndPetDamage(t *testing.T) {
	withTalent := buildHunterForTalentTest(t, 60, "focused_fire", 2, proto.Hunter_Options_Cat)
	withoutTalent := buildHunterForTalentTest(t, 60, "focused_fire", 0, proto.Hunter_Options_Cat)

	wantMult := 1 + 0.02 // rank 2 => 2%
	if got := withTalent.PseudoStats.DamageDealtMultiplier / withoutTalent.PseudoStats.DamageDealtMultiplier; !floatsClose(got, wantMult) {
		t.Errorf("hunter DamageDealtMultiplier ratio = %f, want %f", got, wantMult)
	}
	if got := withTalent.pet.PseudoStats.DamageDealtMultiplier / withoutTalent.pet.PseudoStats.DamageDealtMultiplier; !floatsClose(got, wantMult) {
		t.Errorf("pet DamageDealtMultiplier ratio = %f, want %f", got, wantMult)
	}
}

func TestFocusedFireRequiresAnActivePet(t *testing.T) {
	noPet := buildHunterForTalentTest(t, 60, "focused_fire", 2, proto.Hunter_Options_PetNone)
	if noPet.pet != nil {
		t.Fatal("PetNone built a pet; test setup is wrong")
	}
	if noPet.PseudoStats.DamageDealtMultiplier != 1 {
		t.Errorf("Focused Fire applied with no active pet: DamageDealtMultiplier = %f, want 1", noPet.PseudoStats.DamageDealtMultiplier)
	}
}

func TestLethalAttacksAddsCritToAllAttacks(t *testing.T) {
	withTalent := buildHunterForTalentTest(t, 60, "lethal_attacks", 5, proto.Hunter_Options_PetNone)
	withoutTalent := buildHunterForTalentTest(t, 60, "lethal_attacks", 0, proto.Hunter_Options_PetNone)

	want := 5.0 * core.CritRatingPerCritChance
	if got := withTalent.GetStat(stats.Crit) - withoutTalent.GetStat(stats.Crit); !floatsClose(got, want) {
		t.Errorf("Crit delta = %f, want %f (5%% crit in rating)", got, want)
	}
}

func TestImprovedStingsBuffsSerpentStingDamage(t *testing.T) {
	withTalent := buildHunterForTalentTest(t, 10, "improved_stings", 3, proto.Hunter_Options_PetNone)
	withoutTalent := buildHunterForTalentTest(t, 10, "improved_stings", 0, proto.Hunter_Options_PetNone)

	if withTalent.SerpentSting == nil || withoutTalent.SerpentSting == nil {
		t.Fatal("level-10 hunter has no Serpent Sting registered")
	}

	want := 1.20 // rank 3 => +20%
	if got := withTalent.SerpentSting.DamageMultiplier / withoutTalent.SerpentSting.DamageMultiplier; !floatsClose(got, want) {
		t.Errorf("Serpent Sting DamageMultiplier ratio = %f, want %f", got, want)
	}
}

func TestCarefulAimAddsAttackPowerFromIntellect(t *testing.T) {
	withTalent := buildHunterForTalentTest(t, 60, "careful_aim", 5, proto.Hunter_Options_PetNone)
	withoutTalent := buildHunterForTalentTest(t, 60, "careful_aim", 0, proto.Hunter_Options_PetNone)

	intellect := withoutTalent.GetStat(stats.Intellect)
	if intellect <= 0 {
		t.Fatal("bare level-60 hunter has zero Intellect; Careful Aim has nothing to scale from in this test")
	}

	want := intellect * (0.20 * 5) // rank 5 => 100% of Intellect
	if got := withTalent.GetStat(stats.AttackPower) - withoutTalent.GetStat(stats.AttackPower); !floatsClose(got, want) {
		t.Errorf("AttackPower delta = %f, want %f (100%% of %f Intellect)", got, want, intellect)
	}
}

func TestRapidKillingReducesRapidFireCooldown(t *testing.T) {
	withTalent := buildHunterForTalentTest(t, 60, "rapid_killing", 2, proto.Hunter_Options_PetNone)
	withoutTalent := buildHunterForTalentTest(t, 60, "rapid_killing", 0, proto.Hunter_Options_PetNone)

	if withTalent.RapidFire == nil || withoutTalent.RapidFire == nil {
		t.Fatal("level-60 hunter has no Rapid Fire registered")
	}

	wantDelta := -2 * time.Minute // rank 2 => -2 min
	if got := withTalent.RapidFire.CD.Duration - withoutTalent.RapidFire.CD.Duration; got != wantDelta {
		t.Errorf("Rapid Fire cooldown delta = %v, want %v", got, wantDelta)
	}
}

func TestLoneWolfBuffsDamageOnlyWithoutAPet(t *testing.T) {
	noPet := buildHunterForTalentTest(t, 60, "lone_wolf", 1, proto.Hunter_Options_PetNone)
	if got, want := noPet.PseudoStats.DamageDealtMultiplier, 1.2; !floatsClose(got, want) {
		t.Errorf("no-pet DamageDealtMultiplier = %f, want %f", got, want)
	}

	withPet := buildHunterForTalentTest(t, 60, "lone_wolf", 1, proto.Hunter_Options_Cat)
	if got, want := withPet.PseudoStats.DamageDealtMultiplier, 1.0; !floatsClose(got, want) {
		t.Errorf("with-pet DamageDealtMultiplier = %f, want %f (Lone Wolf must not apply)", got, want)
	}
}

func TestTrueshotAuraSetsTheRaidBuffFlag(t *testing.T) {
	talented := buildHunterForTalentTest(t, 60, "trueshot_aura", 1, proto.Hunter_Options_PetNone)
	raidBuffs := &proto.RaidBuffs{}
	talented.AddRaidBuffs(raidBuffs)
	if !raidBuffs.TrueshotAura {
		t.Error("a Trueshot Aura-talented hunter did not set raidBuffs.TrueshotAura")
	}

	untalented := buildHunterForTalentTest(t, 60, "trueshot_aura", 0, proto.Hunter_Options_PetNone)
	raidBuffs = &proto.RaidBuffs{}
	untalented.AddRaidBuffs(raidBuffs)
	if raidBuffs.TrueshotAura {
		t.Error("an untalented hunter set raidBuffs.TrueshotAura")
	}
}

func TestRapidRecuperationGrantsRegenOnSerpentStingHit(t *testing.T) {
	sim, hunter, target := newRunningHunterForTalentTest(t, 60, "rapid_recuperation", 2, proto.Hunter_Options_PetNone, 25, nil)
	if hunter.SerpentSting == nil {
		t.Fatal("level-60 hunter has no Serpent Sting registered")
	}

	if hunter.PseudoStats.SpiritRegenRateCasting != 0 {
		t.Fatal("test setup already has a nonzero SpiritRegenRateCasting")
	}

	result := &core.SpellResult{Outcome: core.OutcomeLanded, Target: target}
	hunter.SerpentSting.SpellCode = SpellCode_HunterSerpentSting // defensive: already set by serpent_sting.go, re-asserted for this test's own clarity
	hunter.OnSpellHitDealt(sim, hunter.SerpentSting, result)

	want := 0.50 // rank 2 => 50%
	if got := hunter.PseudoStats.SpiritRegenRateCasting; !floatsClose(got, want) {
		t.Errorf("SpiritRegenRateCasting after a landed Serpent Sting hit = %f, want %f", got, want)
	}
}

func TestImprovedTrackingBuffsDamageToListedCreatureTypes(t *testing.T) {
	// core.DefaultTargetProtoLvl60's own MobType is Demon, one of the
	// seven listed types, so the pool-not-prefix grant below reaches it.
	withTalent := buildHunterForTalentTest(t, 60, "improved_tracking", 5, proto.Hunter_Options_PetNone)
	withoutTalent := buildHunterForTalentTest(t, 60, "improved_tracking", 0, proto.Hunter_Options_PetNone)

	target := withTalent.Env.Encounter.AllTargetUnits[0]
	if target.MobType != proto.MobType_MobTypeDemon {
		t.Fatalf("test target MobType = %v, want Demon", target.MobType)
	}

	var withMult, withoutMult float64 = 1, 1
	for _, at := range withTalent.AttackTables[target.UnitIndex] {
		withMult = at.DamageDealtMultiplier
		break
	}
	otherTarget := withoutTalent.Env.Encounter.AllTargetUnits[0]
	for _, at := range withoutTalent.AttackTables[otherTarget.UnitIndex] {
		withoutMult = at.DamageDealtMultiplier
		break
	}

	want := 1.05 // rank 5 => +5%
	if got := withMult / withoutMult; !floatsClose(got, want) {
		t.Errorf("DamageDealtMultiplier ratio against a Demon target = %f, want %f", got, want)
	}
}

func TestResourcefulnessReducesTrapAndMeleeManaCost(t *testing.T) {
	withTalent := buildHunterForTalentTest(t, 60, "resourcefulness", 2, proto.Hunter_Options_PetNone)
	withoutTalent := buildHunterForTalentTest(t, 60, "resourcefulness", 0, proto.Hunter_Options_PetNone)

	if withTalent.ExplosiveTrap == nil || withoutTalent.ExplosiveTrap == nil {
		t.Fatal("level-60 hunter has no Explosive Trap registered")
	}

	wantDelta := int32(-60) // rank 2 => -60%
	if got := withTalent.ExplosiveTrap.Cost.Multiplier - withoutTalent.ExplosiveTrap.Cost.Multiplier; got != wantDelta {
		t.Errorf("Explosive Trap Cost.Multiplier delta = %d, want %d", got, wantDelta)
	}
}

// The regen clause is a 30/60% chance on a critical strike (the live
// text; the engine read 50/100% before the hotfix), so rank 2 is no
// longer a guaranteed proc.
func TestResourcefulnessProcChancePerRank(t *testing.T) {
	for rank, want := range map[int32]float64{1: 0.3, 2: 0.6} {
		if got := resourcefulnessProcChance(rank); !floatsClose(got, want) {
			t.Errorf("rank %d proc chance = %v, want %v", rank, got, want)
		}
	}
}

func TestResourcefulnessGrantsRegenOnCritsAtMaxRank(t *testing.T) {
	sim, hunter, target := newRunningHunterForTalentTest(t, 60, "resourcefulness", 2, proto.Hunter_Options_PetNone, 25, nil)

	crit := &core.SpellResult{Outcome: core.OutcomeLanded | core.OutcomeCrit, Target: target}
	anySpell := &core.Spell{}
	procced := false
	for i := 0; i < 100 && !procced; i++ {
		hunter.OnSpellHitDealt(sim, anySpell, crit)
		procced = hunter.PseudoStats.SpiritRegenRateCasting != 0
	}
	if !procced {
		t.Fatal("Resourcefulness never procced over 100 critical strikes at a 60% chance")
	}
	if got, want := hunter.PseudoStats.SpiritRegenRateCasting, 0.5; !floatsClose(got, want) {
		t.Errorf("SpiritRegenRateCasting after a proc = %f, want %f", got, want)
	}
}

func TestExposePreyResetsMongooseBiteOnHuntersMarkTargets(t *testing.T) {
	// Hunter's Mark is registered pre-finalize through the normal
	// Debuffs pipeline (debuffs.go), not by calling core.HuntersMarkAura
	// directly here: that would try to RegisterAura after finalize
	// (sim.PrePull has already run), which core.Unit.RegisterAura
	// forbids.
	sim, hunter, target := newRunningHunterForTalentTest(t, 30, "expose_prey", 2, proto.Hunter_Options_PetNone, 5,
		&proto.Debuffs{HuntersMark: proto.TristateEffect_TristateEffectRegular})
	if hunter.MongooseBite == nil {
		t.Fatal("level-30 hunter has no Mongoose Bite registered")
	}

	if !target.HasActiveAuraWithTag(core.HuntersMarkAuraTag) {
		t.Fatal("target did not carry an active Hunter's Mark aura")
	}

	hunter.MongooseBite.CD.Use(sim) // put it on cooldown so a reset is observable
	if hunter.MongooseBite.CD.IsReady(sim) {
		t.Fatal("test setup: Mongoose Bite should be on cooldown")
	}

	landedMelee := &core.SpellResult{Outcome: core.OutcomeLanded, Target: target}
	meleeSpell := &core.Spell{}

	reset := false
	for i := 0; i < 300 && !reset; i++ {
		hunter.OnSpellHitDealt(sim, meleeSpell, landedMelee)
		reset = hunter.MongooseBite.CD.IsReady(sim)
	}
	if !reset {
		t.Error("Expose Prey at max rank never reset Mongoose Bite's cooldown over 300 landed attacks against a Hunter's Mark target")
	}
}

func TestSurvivalistsDisciplineReducesTrapCooldown(t *testing.T) {
	withTalent := buildHunterForTalentTest(t, 60, "survivalists_discipline", 2, proto.Hunter_Options_PetNone)
	withoutTalent := buildHunterForTalentTest(t, 60, "survivalists_discipline", 0, proto.Hunter_Options_PetNone)

	if withTalent.ExplosiveTrap == nil || withoutTalent.ExplosiveTrap == nil {
		t.Fatal("level-60 hunter has no Explosive Trap registered")
	}

	want := 0.60 // rank 2 => -40%
	if got := withTalent.ExplosiveTrap.CD.Duration.Seconds() / withoutTalent.ExplosiveTrap.CD.Duration.Seconds(); !floatsClose(got, want) {
		t.Errorf("Explosive Trap cooldown ratio = %f, want %f", got, want)
	}
}

func TestSniperShotRegistersOnlyWhenTalented(t *testing.T) {
	talented := buildHunterForTalentTest(t, 60, "sniper_shot", 1, proto.Hunter_Options_PetNone)
	if talented.SniperShot == nil {
		t.Fatal("SniperShot=true did not register Sniper Shot")
	}
	if got, want := talented.SniperShot.Cost.BaseCost, sniperShotManaCost; got != want {
		t.Errorf("Sniper Shot mana cost = %f, want %f", got, want)
	}

	untalented := buildHunterForTalentTest(t, 60, "sniper_shot", 0, proto.Hunter_Options_PetNone)
	if untalented.SniperShot != nil {
		t.Error("an untalented hunter registered Sniper Shot")
	}
}

func TestSummonHawkRegistersOnlyWhenTalented(t *testing.T) {
	talented := buildHunterForTalentTest(t, 60, "summon_hawk", 1, proto.Hunter_Options_PetNone)
	if talented.SummonHawk == nil {
		t.Fatal("SummonHawk=true did not register Summon Hawk")
	}
	if got, want := talented.SummonHawk.Cost.BaseCost, summonHawkManaCost; got != want {
		t.Errorf("Summon Hawk mana cost = %f, want %f", got, want)
	}

	untalented := buildHunterForTalentTest(t, 60, "summon_hawk", 0, proto.Hunter_Options_PetNone)
	if untalented.SummonHawk != nil {
		t.Error("an untalented hunter registered Summon Hawk")
	}
}

// floatsClose keeps every ratio/delta assertion above from being
// sensitive to float64 rounding across two full character builds.
func floatsClose(got, want float64) bool {
	const epsilon = 1e-6
	diff := got - want
	return diff > -epsilon && diff < epsilon
}
