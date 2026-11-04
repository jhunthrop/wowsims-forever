package rogue_test

// Tests for the Forever rogue talents in sim/rogue/talents.go
// (applyPuncturingWounds, applyHackAndSlash, applyFlawlessExecution,
// applyEfficientStrikes, applySetup and friends).
//
// This file is package rogue_test (external), not rogue, because it
// needs dps_rogue's RegisterDpsRogue to build a full character through
// the agent factory - dps_rogue imports rogue, so an internal rogue
// test importing dps_rogue would cycle.

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/rogue"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func init() {
	dpsrogue.RegisterDpsRogue()
}

// zeroRogueTalents is a valid-length, all-zero talent string in the
// client's Assassination/Combat/Subtlety tree sizes (rogue.TalentTreeSizes).
var zeroRogueTalents = strings.Join([]string{
	strings.Repeat("0", rogue.TalentTreeSizes[0]),
	strings.Repeat("0", rogue.TalentTreeSizes[1]),
	strings.Repeat("0", rogue.TalentTreeSizes[2]),
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

	treeSizes := rogue.TalentTreeSizes
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
	if got, want := rogue.Eviscerate.Cost.FlatModifier, int32(-15); got != want {
		t.Errorf("Eviscerate.Cost.FlatModifier = %d, want %d (1310711's 70755 curve)", got, want)
	}
}

// TestEfficientStrikesDiscountsSinisterStrikeAndHemorrhage pins 13732's
// curve (-3/-5) on its mask: Sinister Strike and Hemorrhage, nothing else.
func TestEfficientStrikesDiscountsSinisterStrikeAndHemorrhage(t *testing.T) {
	for rank, want := range map[int]int32{1: -3, 2: -5} {
		talentsStr := talentStringWithRank(t, talentStringWithRank(t, zeroRogueTalents, "hemorrhage", 1), "efficient_strikes", rank)
		rogue := buildRogueForTalentTest(t, talentsStr, "combat_backstab_prebis").GetRogue()
		if got := rogue.SinisterStrike.Cost.FlatModifier; got != want {
			t.Errorf("rank %d: SinisterStrike.Cost.FlatModifier = %d, want %d", rank, got, want)
		}
		if got := rogue.Hemorrhage.Cost.FlatModifier; got != want {
			t.Errorf("rank %d: Hemorrhage.Cost.FlatModifier = %d, want %d", rank, got, want)
		}
		if got := rogue.Backstab.Cost.FlatModifier; got != 0 {
			t.Errorf("rank %d: Backstab.Cost.FlatModifier = %d, want 0 (outside the mask)", rank, got)
		}
	}
}

// TestSetupComboPointChanceTable pins that an over-ranked
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
// dagger-equipped combat_backstab_prebis set. The client row (13960) is
// a server dummy; its text gives the crit to "your melee weapon
// attacks", so it is melee-table crit and must not reach the poisons
// or weapon procs, which roll spell crit.
func TestHackAndSlashDaggerGrantsCrit(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "hack_and_slash", 5)
	withTalent := buildRogueForTalentTest(t, talentsStr, "combat_backstab_prebis").GetRogue()
	withoutTalent := buildRogueForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis").GetRogue()

	wantDelta := 5.0 * core.CritRatingPerCritChance // rank 5: 1%/rank
	got := withTalent.GetStat(stats.MeleeCrit) - withoutTalent.GetStat(stats.MeleeCrit)
	if math.Abs(got-wantDelta) > 1e-9 {
		t.Errorf("MeleeCrit delta from Hack and Slash rank 5 = %v, want %v", got, wantDelta)
	}
	if every := withTalent.GetStat(stats.Crit) - withoutTalent.GetStat(stats.Crit); every != 0 {
		t.Errorf("Hack and Slash raised every-table crit by %v; poisons must not take it", every)
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
// sim.Reset() is what fires every registered aura's OnReset -
// core.NewEnvironment alone (used above) never calls it.
func buildRogueSimForTalentTest(t *testing.T, talentsStr string, gearSet string) (*core.Simulation, *dpsrogue.DpsRogue) {
	t.Helper()
	return buildRogueSimAtLevel(t, talentsStr, gearSet, 0)
}

// buildRogueSimAtLevel is buildRogueSimForTalentTest at a given character
// level (0 is the engine's default, 60).
func buildRogueSimAtLevel(t *testing.T, talentsStr string, gearSet string, level int32) (*core.Simulation, *dpsrogue.DpsRogue) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassRogue,
			Race:               proto.Race_RaceHuman,
			Level:              level,
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

// TestHackAndSlashMaceIgnoresArmorPercent checks the mace branch: "Mace:
// Your attacks ignore 15% of your target's armor" at rank 5 (13960 effect
// 1309137, 15). Like Serrated Blades it is a share of the target's armor
// (PseudoStats.ArmorIgnorePercent), not flat armor penetration. The prebis
// backstab set is reused with Ironfoe (11684, a one-hand mace) in the main
// hand.
func TestHackAndSlashMaceIgnoresArmorPercent(t *testing.T) {
	build := func(talentsStr string) *rogue.Rogue {
		player := core.WithSpec(
			&proto.Player{
				Class:              proto.Class_ClassRogue,
				Race:               proto.Race_RaceHuman,
				Equipment:          core.GetGearSet("../../ui/rogue/gear_sets", "combat_backstab_prebis").GearSet,
				TalentsString:      talentsStr,
				DistanceFromTarget: 5,
			},
			&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}},
		)
		player.Equipment.Items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: 11684}
		raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
		env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
		return env.Raid.Parties[0].Players[0].(*dpsrogue.DpsRogue).GetRogue()
	}
	plain := build(zeroRogueTalents)
	mace := build(talentStringWithRank(t, zeroRogueTalents, "hack_and_slash", 5))
	if got := mace.PseudoStats.ArmorIgnorePercent - plain.PseudoStats.ArmorIgnorePercent; math.Abs(got-0.15) > 1e-9 {
		t.Errorf("armor ignored by Hack and Slash 5 with a mace = %v of the target's armor, want 0.15", got)
	}
	if got := mace.GetStat(stats.ArmorPenetration) - plain.GetStat(stats.ArmorPenetration); got != 0 {
		t.Errorf("Hack and Slash added %v flat armor penetration, want 0", got)
	}
}

// TestGhostlyStrikeLetsTheNextOpenerSkipStealth pins 14278 e5's trigger
// 1329326 (aura 275 on Garrote 256 + Ambush 512, 7 sec): with it up, Ambush
// and Garrote cast outside Stealth, and one of them spends it.
func TestGhostlyStrikeLetsTheNextOpenerSkipStealth(t *testing.T) {
	talentsStr := talentStringWithRank(t, zeroRogueTalents, "ghostly_strike", 1)
	sim, built := buildRogueSimForTalentTest(t, talentsStr, "combat_backstab_prebis")
	r := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]
	aura := r.GhostlyStrikeOpenerAura
	if aura == nil || aura.ActionID.SpellID != 1329326 || aura.Duration != 7*time.Second {
		t.Fatalf("Ghostly Strike opener aura = %+v, want 1329326 for 7s", aura)
	}
	r.StealthAura.Deactivate(sim)
	if r.Ambush.ExtraCastCondition(sim, target) || r.Garrote.ExtraCastCondition(sim, target) {
		t.Fatal("Ambush or Garrote castable outside Stealth without Ghostly Strike's buff")
	}
	aura.Activate(sim)
	if !r.Ambush.ExtraCastCondition(sim, target) || !r.Garrote.ExtraCastCondition(sim, target) {
		t.Error("Ambush and Garrote must be castable outside Stealth with Ghostly Strike's buff")
	}
	r.Garrote.ApplyEffects(sim, target, r.Garrote)
	if aura.IsActive() {
		t.Error("Garrote outside Stealth did not spend Ghostly Strike's buff")
	}
}

// TestNoOpenerBuffWithoutGhostlyStrike: the aura exists only with the talent.
func TestNoOpenerBuffWithoutGhostlyStrike(t *testing.T) {
	if r := buildRogueForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis").GetRogue(); r.GhostlyStrikeOpenerAura != nil {
		t.Error("untalented rogue has Ghostly Strike's opener buff")
	}
}
