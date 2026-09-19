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

// Every ability a talent modifies must carry its ClassSpellMask, or the
// declarative mod in talents.go silently applies to nothing.
//
// This is asserted against the spells a built mage actually registers.
// The version of this test that shipped first read the `1 << iota`
// constants back and checked they were non-zero and pairwise distinct —
// true by construction of an iota block, and green with every
// `ClassSpellMask:` line deleted from frostbolt.go. Tasks 11 and 12's
// declarative-talent design rests on these masks.
func TestFrostSpellsCarryTheirMasksWhenRegistered(t *testing.T) {
	mage := buildMageForTalentTest(t, ForeverFrostTalents)

	// mask bit -> the registered spells carrying it.
	carriers := map[uint64][]string{}
	for _, spell := range mage.GetCharacter().Spellbook {
		if spell.ClassSpellMask == 0 {
			continue
		}
		for bit := uint64(1); bit != 0; bit <<= 1 {
			if spell.ClassSpellMask&bit != 0 {
				carriers[bit] = append(carriers[bit], spell.ActionID.String())
			}
		}
	}

	named := map[string]uint64{
		"Frostbolt":   MageSpellMaskFrostbolt,
		"Ice Lance":   MageSpellMaskIceLance,
		"Blizzard":    MageSpellMaskBlizzard,
		"Ice Barrier": MageSpellMaskIceBarrier,
	}
	// Frost Nova and Cone of Cold have a mask bit and a talent that
	// names them (Improved Frost Nova, Improved Cone of Cold — two of
	// the nine documented-inert points in ForeverFrostTalents) but no
	// ability file in this package, so nothing registers them. That is
	// recorded here rather than left as a hole in the loop above: when
	// either lands, this fails and the bit moves into `named`.
	unimplemented := map[string]uint64{
		"Frost Nova":   MageSpellMaskFrostNova,
		"Cone of Cold": MageSpellMaskConeOfCold,
	}
	for name, mask := range unimplemented {
		if len(carriers[mask]) != 0 {
			t.Errorf("%s is now registered (as %v); move its mask into the asserted set", name, carriers[mask])
		}
	}
	for name, mask := range named {
		if len(carriers[mask]) == 0 {
			t.Errorf("no registered mage spell carries %s's mask; every talent mod that names it applies to nothing", name)
		}
		for other, otherMask := range named {
			if name < other && mask == otherMask {
				t.Errorf("%s and %s share mask %#x", name, other, mask)
			}
		}
	}

	// The two Frost groups the talents target: every bit in them must be
	// carried by a registered spell, or the mod binds to nothing. Only
	// the Frost groups are checked, because a Frost mage registers no
	// Fire or Arcane damage spell beyond Fire Blast.
	for name, group := range map[string]uint64{
		"MageSpellMaskFrostDamage": MageSpellMaskFrostDamage,
		"MageSpellMaskFrost":       MageSpellMaskFrost,
	} {
		for bit := uint64(1); bit != 0; bit <<= 1 {
			if group&bit == 0 {
				continue
			}
			// Frostfire Bolt, Frost Nova and Cone of Cold are in the
			// group but have no ability file in this package; the
			// `unimplemented` check above is what watches for them.
			if bit == MageSpellMaskFrostfireBolt || bit == MageSpellMaskFrostNova || bit == MageSpellMaskConeOfCold {
				continue
			}
			if len(carriers[bit]) == 0 {
				t.Errorf("%s names bit %#x, which no registered spell carries", name, bit)
			}
		}
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

	// The three assertions above are over the generated table. Rank 11
	// is also gated at registration by core.IncludeAQ, which is a build
	// constant, so the table can be right and the spellbook still stop
	// at rank 10 - the shape a reviewer raised and this pins.
	mage := buildMageForTalentTest(t, ForeverFrostTalents)
	var registered bool
	for _, spell := range mage.GetCharacter().Spellbook {
		if spell.ActionID.SpellID == FrostboltSpellId[11] {
			registered = true
			break
		}
	}
	if !registered {
		t.Errorf("Frostbolt rank 11 (spell %d) is in the table but not in the registered spellbook", FrostboltSpellId[11])
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

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("constructing a mage with ArcaneMeditation=9 and ImprovedConeOfCold=9 panicked: %v", r)
		}
	}()

	built := buildMageForTalentTest(t, talentsStr)

	if got, want := built.PseudoStats.SpiritRegenRateCasting, arcaneMeditationRegenWhileCasting[len(arcaneMeditationRegenWhileCasting)-1]; got != want {
		t.Errorf("SpiritRegenRateCasting with ArcaneMeditation=9 = %v, want the max-rank value %v", got, want)
	}
}

// The three SpellMod_Threat_Pct talents multiply 1 - perRank*rank, so an
// unvalidated talent string (core.FillTalentsProto does no max-rank
// check) could drive the multiplier negative — rank 9 of Arcane Subtlety
// is 1 - 0.15*9 = -0.35 — and core.removeThreatPct then divides by it.
// rankOf clamps the rank to the talent's own max. A talent string holds
// one digit per node, so 9 is the worst an attacker can write; at
// 0.15/rank Arcane Subtlety inverts there, and the other two "only"
// reduce threat by 90% instead of 30%. The last assertion is what keeps
// this from being a tautology: it fails if the clamp stops changing the
// value, at which point the test needs a new witness rather than a
// quiet pass.
func TestOverRankedThreatTalentsCannotInvertTheThreatMultiplier(t *testing.T) {
	for _, c := range []struct {
		talent  string
		perRank float64
	}{
		{"arcane_subtlety", arcaneSubtletyThreatReductionPerRank},
		{"burning_soul", burningSoulThreatReductionPerRank},
		{"frost_channeling", frostChannelingThreatReductionPerRank},
	} {
		maxRank := int32(len(TalentSpellIDs[c.talent]))
		if maxRank == 0 {
			t.Errorf("TalentSpellIDs has no entry for %q, so rankOf clamps to 0", c.talent)
			continue
		}
		if got := rankOf(c.talent, 9); got != maxRank {
			t.Errorf("rankOf(%q, 9) = %d, want the talent's max rank %d", c.talent, got, maxRank)
		}
		if mult := 1 - c.perRank*float64(rankOf(c.talent, 9)); mult <= 0 {
			t.Errorf("%s over-ranked to 9 leaves a threat multiplier of %v, which must stay positive", c.talent, mult)
		}
		if clamped, unclamped := 1-c.perRank*float64(rankOf(c.talent, 9)), 1-c.perRank*9; clamped == unclamped {
			t.Errorf("%s at rank 9 gives %v clamped and unclamped alike: this test no longer witnesses the clamp", c.talent, clamped)
		}
	}
}

// buildMageForTalentTest stands up one mage through the shipping agent
// factory, in the P1 gear and consumes the regression suite uses, so the
// spells and stats under test are the ones a sim sees.
func buildMageForTalentTest(t *testing.T, talentsStr string) *Mage {
	t.Helper()

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

	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)

	built, ok := env.Raid.Parties[0].Players[0].(*Mage)
	if !ok {
		t.Fatal("player 0 did not build as a *Mage")
	}
	return built
}
