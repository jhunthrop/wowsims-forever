package mage

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
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
