package mage

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// The generated tree must be the client's: Arcane 18, Fire 17, Frost 19,
// from build 1.60.1.69893. Frost is the largest tree in the game, three
// wider than vanilla's, so a [3]int{16, 16, 17} here means the
// hand-written vanilla sizes survived and Task 17 did not run.
func TestTheGeneratedTreeIsTheClients(t *testing.T) {
	if got, want := TalentTreeSizes, [3]int{18, 17, 19}; got != want {
		t.Fatalf("TalentTreeSizes = %v, want %v (Arcane 18, Fire 17, Frost 19)", got, want)
	}
	if TalentsBuild == "" {
		t.Error("TalentsBuild is empty; the generated file must record the client build")
	}
}

// The talent string is parsed positionally against TalentTreeSizes, so a
// mismatch silently reads the wrong talent.
func TestTalentTreeSizesMatchTheProto(t *testing.T) {
	var total int
	for _, n := range TalentTreeSizes {
		total += n
	}
	fields := (&proto.MageTalents{}).ProtoReflect().Descriptor().Fields()
	if fields.Len() != total {
		t.Errorf("MageTalents has %d fields, TalentTreeSizes sums to %d", fields.Len(), total)
	}
	if total != 54 {
		t.Errorf("the mage has %d talents, want 54 from build %s", total, TalentsBuild)
	}
}

// Every talent this spec's behaviour reads must exist in the client's
// tree. Keep foreverFrostTalentsApplied in step with talents.go.
var foreverFrostTalentsApplied = []string{
	// Arcane.
	"arcane_focus",
	"arcane_subtlety",
	"magic_absorption",
	"arcane_concentration",
	"arcane_impact",
	"arcane_meditation",
	"arcane_mind",
	"arcane_instability",
	"presence_of_mind",
	"arcane_power",
	// Fire.
	"incineration",
	"improved_fireball",
	"ignite",
	"burning_soul",
	"improved_flamestrike",
	"improved_scorch",
	"master_of_elements",
	"critical_mass",
	"fire_power",
	"combustion",
	"wake_of_fire",
	// Frost.
	"improved_frostbolt",
	"elemental_precision",
	"ice_shards",
	"improved_frost_nova",
	"piercing_ice",
	"frost_channeling",
	"ice_lance",
	"improved_blizzard",
	"improved_cone_of_cold",
	"cold_snap",
	"winters_chill",
	"ice_barrier",
}

func TestEveryTalentThisSpecAppliesExists(t *testing.T) {
	for _, name := range foreverFrostTalentsApplied {
		if _, ok := TalentNodeIDs[name]; !ok {
			t.Errorf("talents.go applies %q, which is not in the client's tree", name)
		}
		if len(TalentSpellIDs[name]) == 0 {
			t.Errorf("%q has no rank spell ids", name)
		}
	}
}

func TestFrostSpellsCarryTheirMasks(t *testing.T) {
	cases := []struct {
		name string
		mask uint64
	}{
		{"Frostbolt", MageSpellMaskFrostbolt},
		{"Ice Lance", MageSpellMaskIceLance},
		{"Frost Nova", MageSpellMaskFrostNova},
		{"Blizzard", MageSpellMaskBlizzard},
		{"Cone of Cold", MageSpellMaskConeOfCold},
	}
	seen := uint64(0)
	for _, c := range cases {
		if c.mask == 0 {
			t.Errorf("%s has a zero mask; an empty mask matches nothing", c.name)
		}
		if seen&c.mask != 0 {
			t.Errorf("%s reuses a bit already taken", c.name)
		}
		seen |= c.mask
	}
}

// The data lane verified that the engine's checked-in preset APL casts
// Frostbolt rank 10 where the Era tables give rank 11 for spell 25304.
// The Forever APL must use the highest rank the character has.
func TestFrostboltHasElevenRanks(t *testing.T) {
	if FrostboltRanks < 11 {
		t.Fatalf("FrostboltRanks = %d, want at least 11", FrostboltRanks)
	}
	if got := FrostboltSpellId[11]; got != 25304 {
		t.Errorf("FrostboltSpellId[11] = %d, want 25304", got)
	}
	if FrostboltLevel[11] > 60 {
		t.Errorf("Frostbolt rank 11 requires level %d; a level-60 mage cannot cast it", FrostboltLevel[11])
	}
}

func TestForeverFrostTalentsAreAValidBuild(t *testing.T) {
	// Each segment must be exactly its tree's width, or every talent
	// after the short one is read from the wrong position and nothing
	// complains. Frost is 19 wide against vanilla's 16, so a string
	// carried over from an Era build is the likely failure.
	parts := strings.Split(ForeverFrostTalents, "-")
	if len(parts) != 3 {
		t.Fatalf("ForeverFrostTalents has %d segments, want 3", len(parts))
	}
	var spent int
	for i, part := range parts {
		if len(part) != TalentTreeSizes[i] {
			t.Errorf("segment %d is %d characters, want %d", i, len(part), TalentTreeSizes[i])
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				t.Fatalf("segment %d contains %q", i, c)
			}
			spent += int(c - '0')
		}
	}
	if spent != 51 {
		t.Errorf("the reference build spends %d points, want 51", spent)
	}
	if sum(parts[2]) != 31 {
		t.Errorf("the reference build spends %d points in Frost, want 31 to reach the capstone", sum(parts[2]))
	}

	talents := &proto.MageTalents{}
	fillMageTalents(talents, ForeverFrostTalents)
	// Both, not either: the capstone and the deep filler are separate
	// facts and an && between them passes when one is missing.
	if !talents.IceBarrier {
		t.Error("the reference Frost build does not take Ice Barrier, the 31-point Frost talent")
	}
	if talents.WintersChill == 0 {
		t.Error("the reference Frost build puts no points in Winter's Chill")
	}
}

func sum(segment string) int {
	var n int
	for _, c := range segment {
		n += int(c - '0')
	}
	return n
}

// rankIndex clamps a table lookup to the table's max rank rather than
// indexing out of range, because core.FillTalentsProto does not
// validate a talent string against the client's per-node max rank.
func TestRankIndexClampsAnOverRankToTheTablesMaxEntry(t *testing.T) {
	if got, want := improvedConeOfColdDamage[rankIndex(9, improvedConeOfColdDamage[:])], improvedConeOfColdDamage[len(improvedConeOfColdDamage)-1]; got != want {
		t.Errorf("improvedConeOfColdDamage at rank 9 = %d, want the max-rank value %d", got, want)
	}
	if got, want := arcaneMeditationRegenWhileCasting[rankIndex(9, arcaneMeditationRegenWhileCasting[:])], arcaneMeditationRegenWhileCasting[len(arcaneMeditationRegenWhileCasting)-1]; got != want {
		t.Errorf("arcaneMeditationRegenWhileCasting at rank 9 = %v, want the max-rank value %v", got, want)
	}
	// A rank inside the table still reads its own entry, not the max.
	if got, want := improvedConeOfColdDamage[rankIndex(1, improvedConeOfColdDamage[:])], improvedConeOfColdDamage[1]; got != want {
		t.Errorf("improvedConeOfColdDamage at rank 1 = %d, want %d", got, want)
	}
}

// talentStringWithRank returns a copy of talentsStr with the named
// MageTalents field's digit set to rank, found positionally the same
// way core.FillTalentsProto reads it (proto field number against the
// tree-segment offsets in TalentTreeSizes).
func talentStringWithRank(t *testing.T, talentsStr string, fieldName string, rank int) string {
	t.Helper()

	fd := (&proto.MageTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if fd == nil {
		t.Fatalf("MageTalents has no field named %q", fieldName)
	}

	pos := int(fd.Number()) - 1
	treeIdx := 0
	for treeIdx < len(TalentTreeSizes) && pos >= TalentTreeSizes[treeIdx] {
		pos -= TalentTreeSizes[treeIdx]
		treeIdx++
	}

	parts := strings.Split(talentsStr, "-")
	chars := []rune(parts[treeIdx])
	chars[pos] = rune('0' + rank)
	parts[treeIdx] = string(chars)
	return strings.Join(parts, "-")
}

// A talent string with more points in a talent than the client allows
// (a corrupt or hand-edited string, since FillTalentsProto does not
// validate against the per-node max rank) must clamp to the talent's
// max rank rather than panic with an index out of range.
func TestMageWithOverRankTalentsConstructsWithoutPanicking(t *testing.T) {
	talentsStr := talentStringWithRank(t, ForeverFrostTalents, "arcane_meditation", 9)
	talentsStr = talentStringWithRank(t, talentsStr, "improved_cone_of_cold", 9)

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Equipment:          core.GetGearSet("../../ui/mage/gear_sets", "p0.bis").GearSet,
			Consumes:           P1Consumes.Consumes,
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
		},
		PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("constructing a mage with ArcaneMeditation=9 and ImprovedConeOfCold=9 panicked: %v", r)
		}
	}()

	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)

	built, ok := env.Raid.Parties[0].Players[0].(*Mage)
	if !ok {
		t.Fatal("player 0 did not build as a *Mage")
	}

	if got, want := built.PseudoStats.SpiritRegenRateCasting, arcaneMeditationRegenWhileCasting[len(arcaneMeditationRegenWhileCasting)-1]; got != want {
		t.Errorf("SpiritRegenRateCasting with ArcaneMeditation=9 = %v, want the max-rank value %v", got, want)
	}
}
