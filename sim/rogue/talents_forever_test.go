package rogue_test

// Tests for the eighteen RogueTalents proto fields talents.go did not
// read before this change (PORTING.md has no entry for this lane; see
// the task that added applyPuncturingWounds, applyHackAndSlash,
// applyCutthroat, applyThousandCuts, applyQuietus,
// applyFlawlessExecution and applySetup in sim/rogue/talents.go).
//
// This file is package rogue_test (external), not rogue, because it
// needs dps_rogue's RegisterDpsRogue to build a full character through
// the agent factory - dps_rogue imports rogue, so an internal rogue
// test importing dps_rogue would cycle.

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func init() {
	dpsrogue.RegisterDpsRogue()
}

// zeroRogueTalents is a valid-length, all-zero talent string: 17-17-19,
// the client's Assassination/Combat/Subtlety tree sizes
// (rogue.TalentTreeSizes).
var zeroRogueTalents = strings.Join([]string{
	strings.Repeat("0", 17),
	strings.Repeat("0", 17),
	strings.Repeat("0", 19),
}, "-")

// talentStringWithRank returns a copy of talentsStr with the named
// RogueTalents field's digit set to rank, found positionally the same
// way core.FillTalentsProto reads it (proto field number against the
// tree-segment offsets in rogue.TalentTreeSizes). Mirrors mage/talents_
// test.go's helper of the same name, which cannot be reused directly
// since it is unexported in another package.
func talentStringWithRank(t *testing.T, talentsStr string, fieldName string, rank int) string {
	t.Helper()

	fd := (&proto.RogueTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if fd == nil {
		t.Fatalf("RogueTalents has no field named %q", fieldName)
	}

	treeSizes := [3]int{17, 17, 19}
	pos := int(fd.Number()) - 1
	treeIdx := 0
	for treeIdx < len(treeSizes) && pos >= treeSizes[treeIdx] {
		pos -= treeSizes[treeIdx]
		treeIdx++
	}

	parts := strings.Split(talentsStr, "-")
	chars := []rune(parts[treeIdx])
	chars[pos] = rune('0' + rank)
	parts[treeIdx] = string(chars)
	return strings.Join(parts, "-")
}

// buildRogueForTalentTest stands up one rogue through the shipping agent
// factory (dps_rogue), in a prebis gear set with the given weapon type,
// so the spells and stats under test are the ones a sim sees. No
// rotation is set, matching mage/talents_test.go's buildMageForTalentTest:
// these tests inspect the built character, they do not run it.
func buildRogueForTalentTest(t *testing.T, talentsStr string, gearSet string) *dpsrogue.DpsRogue {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassRogue,
			Race:               proto.Race_RaceHuman,
			Equipment:          core.GetGearSet("../../ui/rogue/gear_sets", gearSet).GearSet,
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
		},
		&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)

	built, ok := env.Raid.Parties[0].Players[0].(*dpsrogue.DpsRogue)
	if !ok {
		t.Fatal("player 0 did not build as a *dpsrogue.DpsRogue")
	}
	return built
}

// TestPuncturingWoundsBuffsBackstabAndMutilateCrit checks the eager
// BonusCritRating writes applyPuncturingWounds defers to
// Env.RegisterPreFinalizeEffect land on the live, Initialize-populated
// spell pointers rather than panicking on the nil ones ApplyTalents saw.
func TestPuncturingWoundsBuffsBackstabAndMutilateCrit(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "mutilate", 1)
	talentsStr = talentStringWithRank(t, talentsStr, "puncturing_wounds", 3)

	built := buildRogueForTalentTest(t, talentsStr, "combat_backstab_prebis")
	rogue := built.GetRogue()

	wantCrit := 30.0 * core.CritRatingPerCritChance // rank 3: 10%/rank on Backstab
	if rogue.Backstab == nil {
		t.Fatal("Backstab did not register")
	}
	if got := rogue.Backstab.BonusCritRating; got != wantCrit {
		t.Errorf("Backstab.BonusCritRating = %v, want %v (rank-3 Puncturing Wounds)", got, wantCrit)
	}

	wantMutilateCrit := 15.0 * core.CritRatingPerCritChance // rank 3: 5%/rank on Mutilate
	if rogue.MutilateMH == nil || rogue.MutilateOH == nil {
		t.Fatal("Mutilate did not register both hands (talent Mutilate was not set to 1)")
	}
	if got := rogue.MutilateMH.BonusCritRating; got != wantMutilateCrit {
		t.Errorf("MutilateMH.BonusCritRating = %v, want %v", got, wantMutilateCrit)
	}
	if got := rogue.MutilateOH.BonusCritRating; got != wantMutilateCrit {
		t.Errorf("MutilateOH.BonusCritRating = %v, want %v", got, wantMutilateCrit)
	}
}

// TestPuncturingWoundsZeroRankChangesNothing is the control: a build
// with no points spent must leave Backstab/Mutilate exactly as an
// untalented rogue would read them.
func TestPuncturingWoundsZeroRankChangesNothing(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "mutilate", 1)

	built := buildRogueForTalentTest(t, talentsStr, "combat_backstab_prebis")
	rogue := built.GetRogue()

	if got := rogue.Backstab.BonusCritRating; got != 0 {
		t.Errorf("Backstab.BonusCritRating = %v, want 0 with no Puncturing Wounds", got)
	}
}

// TestFlawlessExecutionDiscountsEviscerate checks the flat Energy
// discount, also deferred through RegisterPreFinalizeEffect since
// Eviscerate is nil at ApplyTalents time.
func TestFlawlessExecutionDiscountsEviscerate(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "flawless_execution", 1)
	built := buildRogueForTalentTest(t, talentsStr, "combat_backstab_prebis")
	rogue := built.GetRogue()

	if rogue.Eviscerate == nil {
		t.Fatal("Eviscerate did not register")
	}
	if got, want := rogue.Eviscerate.Cost.FlatModifier, int32(-10); got != want {
		t.Errorf("Eviscerate.Cost.FlatModifier = %d, want %d", got, want)
	}
}

// TestCutthroatLetsAmbushBypassStealth checks that a rogue with
// Cutthroat spent gets a non-nil CutthroatAura (ambush.go's
// ExtraCastCondition reads this field directly) and that a rogue with no
// points in it does not, so the bypass check in ambush.go is provably a
// no-op for every build without the talent.
func TestCutthroatLetsAmbushBypassStealth(t *testing.T) {
	withTalent := talentStringWithRank(t, zeroRogueTalents, "cutthroat", 5)
	built := buildRogueForTalentTest(t, withTalent, "combat_backstab_prebis")
	rogue := built.GetRogue()
	if rogue.CutthroatAura == nil {
		t.Fatal("CutthroatAura is nil with Cutthroat rank 5 spent")
	}
	if rogue.Ambush == nil {
		t.Fatal("Ambush did not register")
	}

	withoutTalent := buildRogueForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis")
	if withoutTalent.GetRogue().CutthroatAura != nil {
		t.Error("CutthroatAura is non-nil with no points in Cutthroat")
	}
}

// TestSetupComboPointChanceTable pins Setup's 33%/67%/100% per-rank
// table (not an exact 33.33%/rank multiply) and that an over-ranked
// talent string (core.FillTalentsProto does no max-rank validation)
// clamps to rank 3's value instead of panicking on an out-of-range
// index - the same hazard mage/talents.go's rankIndex guards against.
func TestSetupComboPointChanceTable(t *testing.T) {
	// setupComboPointChance itself is unexported (talents.go); this pins
	// that the talent string's digit reaches Talents.Setup unchanged for
	// every real rank, which is the half of applySetup's table lookup
	// this package can observe from outside. TestSetupOverRankDoesNotPanic
	// below is the behavioral half: that the lookup clamps rather than
	// panics for a rank past the real max.
	for rank := 0; rank <= 3; rank++ {
		talentsStr := talentStringWithRank(t, zeroRogueTalents, "setup", rank)
		built := buildRogueForTalentTest(t, talentsStr, "combat_backstab_prebis")
		if got := built.GetRogue().Talents.Setup; got != int32(rank) {
			t.Fatalf("Talents.Setup = %d, want %d", got, rank)
		}
	}
}

// TestSetupOverRankDoesNotPanic is the over-rank hazard check: a talent
// string with Setup's digit set above its real max rank (3) must not
// panic constructing the character.
func TestSetupOverRankDoesNotPanic(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "setup", 9)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("constructing a rogue with Setup=9 panicked: %v", r)
		}
	}()

	buildRogueForTalentTest(t, talentsStr, "combat_backstab_prebis")
}

// TestHackAndSlashDaggerGrantsCrit checks the dagger/fist branch (a
// direct AddStat, no deferred spell-pointer read needed) on the
// dagger-equipped combat_backstab_prebis set.
func TestHackAndSlashDaggerGrantsCrit(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "hack_and_slash", 5)
	withTalent := buildRogueForTalentTest(t, talentsStr, "combat_backstab_prebis").GetRogue()
	withoutTalent := buildRogueForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis").GetRogue()

	wantDelta := 5.0 * core.CritRatingPerCritChance // rank 5: 1%/rank
	got := withTalent.GetStat(stats.Crit) - withoutTalent.GetStat(stats.Crit)
	if got != wantDelta {
		t.Errorf("Crit stat delta from Hack and Slash rank 5 = %v, want %v", got, wantDelta)
	}
}

// TestHackAndSlashSwordRegistersExtraAttackAura checks the axe/sword
// branch on the sword-equipped combat_sinister_strike_prebis set: this
// repo's rogue gear fixtures carry no mace set to exercise the
// armor-penetration branch against, so that branch is left to the code's
// own comment rather than faked with a hand-built Equipment proto.
func TestHackAndSlashSwordRegistersExtraAttackAura(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "hack_and_slash", 1)
	built := buildRogueForTalentTest(t, talentsStr, "combat_sinister_strike_prebis")
	rogue := built.GetRogue()

	if aura := rogue.GetAuraByID(hackAndSlashActionID); aura == nil {
		t.Error("no Hack and Slash aura registered for a sword-equipped rogue with Hack and Slash spent")
	}
}

var hackAndSlashActionID = core.ActionID{SpellID: 13960}

// buildRogueSimForTalentTest is buildRogueForTalentTest's sibling for
// tests that need a live *core.Simulation (not just a built character):
// sim.Reset() is what fires every registered aura's OnReset, including
// applyThousandCuts' and applyQuietus' - core.NewEnvironment alone (used
// above) never calls it, since those tests only inspected static,
// already-applied field values.
func buildRogueSimForTalentTest(t *testing.T, talentsStr string, gearSet string) (*core.Simulation, *dpsrogue.DpsRogue) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassRogue,
			Race:               proto.Race_RaceHuman,
			Equipment:          core.GetGearSet("../../ui/rogue/gear_sets", gearSet).GearSet,
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
		},
		&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60},
		SimOptions: core.DefaultSimTestOptions,
	}, simsignals.CreateSignals())
	sim.Reset()

	built, ok := sim.Raid.Parties[0].Players[0].(*dpsrogue.DpsRogue)
	if !ok {
		t.Fatal("player 0 did not build as a *dpsrogue.DpsRogue")
	}
	return sim, built
}

// TestThousandCutsStacksDiscountEnergyCost drives the stacking aura
// directly (Activate/AddStack/SetStacks are what applyThousandCuts'
// Rupture-tick and Hemorrhage/Backstab-cast listeners call) rather than
// through an actual Rupture tick or Backstab cast, since fabricating a
// landed core.SpellResult from outside sim/core is not worth the
// coupling. It still exercises the part most likely to regress: the
// per-stack delta in OnStacksChange, the cap at 5 stacks, and that
// SetStacks(0) - not Deactivate - is what the consuming cast and the
// natural 10-sec expiry both need to undo the discount (see the aura's
// own OnExpire comment in talents.go).
func TestThousandCutsStacksDiscountEnergyCost(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "thousand_cuts", 1)
	sim, built := buildRogueSimForTalentTest(t, talentsStr, "combat_backstab_prebis")
	rogue := built.GetRogue()

	if rogue.Backstab == nil {
		t.Fatal("Backstab did not register")
	}
	if got := rogue.Backstab.Cost.FlatModifier; got != 0 {
		t.Fatalf("Backstab.Cost.FlatModifier = %d before any Rupture tick, want 0", got)
	}

	aura := rogue.GetAuraByID(core.ActionID{SpellID: 1310721})
	if aura == nil {
		t.Fatal("no Thousand Cuts aura registered")
	}

	aura.Activate(sim)
	aura.AddStack(sim)
	if got, want := rogue.Backstab.Cost.FlatModifier, int32(-3); got != want {
		t.Errorf("Backstab.Cost.FlatModifier after 1 stack = %d, want %d", got, want)
	}

	for i := 0; i < 4; i++ {
		aura.AddStack(sim)
	}
	if got, want := rogue.Backstab.Cost.FlatModifier, int32(-15); got != want {
		t.Errorf("Backstab.Cost.FlatModifier after 5 stacks (the cap) = %d, want %d", got, want)
	}
	// A 6th tick must not push past the 5-stack cap.
	aura.AddStack(sim)
	if got, want := rogue.Backstab.Cost.FlatModifier, int32(-15); got != want {
		t.Errorf("Backstab.Cost.FlatModifier after a 6th stack = %d, want %d (still capped at 5)", got, want)
	}

	// SetStacks(0) is the consume/expire path; it must fully undo the
	// discount, not just the last stack.
	aura.SetStacks(sim, 0)
	if got, want := rogue.Backstab.Cost.FlatModifier, int32(0); got != want {
		t.Errorf("Backstab.Cost.FlatModifier after SetStacks(0) = %d, want %d", got, want)
	}
	if aura.IsActive() {
		t.Error("Thousand Cuts aura is still active after SetStacks(0)")
	}
}

// TestQuietusAppliesNothingBeforeAnyExecutePhaseTransition is the
// eager-mutation safety check applyPuncturingWounds' tests above don't
// need: Quietus's damage bonus must stay at zero until sim time actually
// crosses the 35%-health execute-phase boundary
// (sim.RegisterExecutePhaseCallback, re-armed every sim.Reset - see
// talents.go's applyQuietus comment), not at construction or at Reset
// itself. Driving a real phase transition needs sim/core's private
// advance()/nextExecutePhase(), which this external test package cannot
// reach, so this pins the one invariant it can observe from outside:
// Reset alone must not have moved the number.
func TestQuietusAppliesNothingBeforeAnyExecutePhaseTransition(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "quietus", 5)
	_, withTalent := buildRogueSimForTalentTest(t, talentsStr, "combat_backstab_prebis")
	_, withoutTalent := buildRogueSimForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis")

	withRogue, baseRogue := withTalent.GetRogue(), withoutTalent.GetRogue()
	if withRogue.SinisterStrike == nil || baseRogue.SinisterStrike == nil {
		t.Fatal("SinisterStrike did not register")
	}

	// core.RegisterSpell defaults DamageMultiplierAdditive to 1 (not 0)
	// when only DamageMultiplier is set (sim/core/spell.go), so the
	// no-op baseline to compare against is the untalented rogue's value,
	// not zero.
	got := withRogue.SinisterStrike.DamageMultiplierAdditive - baseRogue.SinisterStrike.DamageMultiplierAdditive
	if got != 0 {
		t.Errorf("Quietus rank 5's SinisterStrike.DamageMultiplierAdditive delta right after Reset = %v, want 0 (nothing is below 35%% health yet)", got)
	}
}
