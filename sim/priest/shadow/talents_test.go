package shadow

import (
	"strings"
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get caster sets included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/priest"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// allZeroPriestTalents is three dash-joined segments of '0', sized to
// priest.TalentTreeSizes (Discipline, Holy, Shadow), so a test can set a
// single named talent's rank without any other talent's behaviour
// mixing in - unlike P1Talents, which predates the client talent-tree
// rewrite (SkipAwaitingForeverTalentRewrite in shadow_priest_test.go)
// and reads garbage ranks for fields added since, such as
// DevouringContagion (see priest/talents.go's rankOf comment).
func allZeroPriestTalents() string {
	segments := make([]string, len(priest.TalentTreeSizes))
	for i, n := range priest.TalentTreeSizes {
		segments[i] = strings.Repeat("0", n)
	}
	return strings.Join(segments, "-")
}

// talentStringWithRank returns a copy of talentsStr with the named
// PriestTalents field's digit set to rank, found positionally the same
// way core.FillTalentsProto reads it (proto field number against the
// tree-segment offsets in priest.TalentTreeSizes) - mirroring
// sim/mage/talents_test.go's helper of the same name.
func talentStringWithRank(t *testing.T, talentsStr string, fieldName string, rank int) string {
	t.Helper()

	fd := (&proto.PriestTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if fd == nil {
		t.Fatalf("PriestTalents has no field named %q", fieldName)
	}

	pos := int(fd.Number()) - 1
	treeIdx := 0
	for treeIdx < len(priest.TalentTreeSizes) && pos >= priest.TalentTreeSizes[treeIdx] {
		pos -= priest.TalentTreeSizes[treeIdx]
		treeIdx++
	}

	parts := strings.Split(talentsStr, "-")
	chars := []rune(parts[treeIdx])
	chars[pos] = rune('0' + rank)
	parts[treeIdx] = string(chars)
	return strings.Join(parts, "-")
}

// buildShadowPriestForTalentTest stands up one Undead shadow priest
// (Devouring Plague requires Undead - priest.Initialize) at max level
// with the given talents string, through the shipping agent factory, so
// the spells under test are the ones a real sim registers.
func buildShadowPriestForTalentTest(t *testing.T, talentsStr string) *ShadowPriest {
	t.Helper()

	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassPriest,
		Race:          proto.Race_RaceUndead,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		Buffs:         core.FullBuffs.Player,
		TalentsString: talentsStr,
	}, PlayerOptionsBasic)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}

	env, _, _ := core.NewEnvironment(raid, encounter, true)
	built, ok := env.Raid.Parties[0].Players[0].(*ShadowPriest)
	if !ok {
		t.Fatal("player 0 did not build as a *ShadowPriest")
	}
	return built
}

// Twin Disciplines at rank 5 ("+5% instant cast spell damage") must
// raise DamageMultiplier on every instant-cast Shadow damage spell this
// package registers, and must leave Mind Blast - a cast-time spell -
// untouched.
func TestTwinDisciplinesRaisesInstantShadowSpellDamage(t *testing.T) {
	talents := talentStringWithRank(t, allZeroPriestTalents(), "twin_disciplines", 5)
	// Mind Flay is itself a bool talent (registerMindFlay returns early
	// at rank 0); take it so Mind Flay is in the spellbook to assert on.
	talents = talentStringWithRank(t, talents, "mind_flay", 1)
	spriest := buildShadowPriestForTalentTest(t, talents)

	wantMultiplier := 1.05
	for name, spell := range map[string]*core.Spell{
		"Mind Flay (rank 6, tick 0)":  spriest.MindFlay[priest.MindFlayRanks][0],
		"Devouring Plague (top rank)": spriest.DevouringPlague[priest.DevouringPlagueRanks],
		"Shadow Word: Pain (rank 8)":  spriest.ShadowWordPain[priest.ShadowWordPainRanks],
		"Shadow Word: Death (rank 4)": spriest.ShadowWordDeath[len(spriest.ShadowWordDeath)-1],
	} {
		if spell == nil {
			t.Fatalf("%s not registered", name)
		}
		if got := spell.DamageMultiplier; got != wantMultiplier {
			t.Errorf("%s DamageMultiplier = %v, want %v", name, got, wantMultiplier)
		}
	}

	if got, want := spriest.MindBlast[len(spriest.MindBlast)-1].DamageMultiplier, 1.0; got != want {
		t.Errorf("Mind Blast (cast-time, not instant) DamageMultiplier = %v, want %v untouched by Twin Disciplines", got, want)
	}
}

// Improved Mind Flay at rank 2 ("+20% Mind Flay damage") must raise
// DamageMultiplier on every Mind Flay tick spell and leave Shadow Word:
// Pain untouched.
func TestImprovedMindFlayRaisesMindFlayDamageOnly(t *testing.T) {
	talents := talentStringWithRank(t, allZeroPriestTalents(), "improved_mind_flay", 2)
	talents = talentStringWithRank(t, talents, "mind_flay", 1)
	spriest := buildShadowPriestForTalentTest(t, talents)

	for tick, spell := range spriest.MindFlay[priest.MindFlayRanks] {
		if spell == nil {
			continue
		}
		if got, want := spell.DamageMultiplier, 1.20; got != want {
			t.Errorf("Mind Flay tick %d DamageMultiplier = %v, want %v", tick, got, want)
		}
	}

	if got, want := spriest.ShadowWordPain[priest.ShadowWordPainRanks].DamageMultiplier, 1.0; got != want {
		t.Errorf("Shadow Word: Pain DamageMultiplier = %v, want %v untouched by Improved Mind Flay", got, want)
	}
}

// Devouring Contagion at rank 2 ("-50% Devouring Plague mana cost") must
// lower Devouring Plague's Cost.Multiplier by exactly 50 and leave Mind
// Flay's cost untouched. This is also the regression witness for
// priest/talents.go's rankOf clamp: without it, sim/priest/shadow's own
// (stale) P1Talents reads this field as rank 5 and drives the mana cost
// negative - TestDevouringPlagueRank6ResolvesDistinctlyFromRank5ViaGetSpell
// caught that live.
func TestDevouringContagionLowersDevouringPlagueCostOnly(t *testing.T) {
	talents := talentStringWithRank(t, allZeroPriestTalents(), "devouring_contagion", 2)
	talents = talentStringWithRank(t, talents, "mind_flay", 1)
	spriest := buildShadowPriestForTalentTest(t, talents)

	dp := spriest.DevouringPlague[priest.DevouringPlagueRanks]
	if dp == nil || dp.Cost == nil {
		t.Fatal("Devouring Plague top rank not registered, or has no Cost")
	}
	if got, want := dp.Cost.Multiplier, int32(50); got != want {
		t.Errorf("Devouring Plague Cost.Multiplier = %d, want %d (100 - 50)", got, want)
	}
	if got := dp.Cost.GetCurrentCost(); got <= 0 {
		t.Errorf("Devouring Plague GetCurrentCost() = %v, want a positive mana cost", got)
	}

	mf := spriest.MindFlay[priest.MindFlayRanks][0]
	if mf == nil || mf.Cost == nil {
		t.Fatal("Mind Flay top rank not registered, or has no Cost")
	}
	if got, want := mf.Cost.Multiplier, int32(100); got != want {
		t.Errorf("Mind Flay Cost.Multiplier = %d, want %d untouched by Devouring Contagion", got, want)
	}
}
