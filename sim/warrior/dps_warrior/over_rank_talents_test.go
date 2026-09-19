package dpswarrior

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// core.FillTalentsProto reads one character per talent node, discards the
// strconv error and writes the value straight into the proto with no
// max-rank validation, and the envelope's Validate never looks at Talents
// either — so a corrupt or hand-edited talent string arriving from the web
// reaches this package's rank tables raw. Every one of them is now read
// through warrior.rankIndex, which clamps to the table's top rank; before
// that, an over-ranked string was an `index out of range` panic that the
// engine's runSim recover turned into a Go runtime message shown to the
// user. This is the warrior twin of
// TestMageWithOverRankTalentsConstructsWithoutPanicking: the mage half of
// the bug was fixed when rankIndex was written and the warrior half was
// not.
//
// Every talent the warrior reads through a rank table is over-ranked here
// at once, so a new table added without a clamp fails as soon as its
// talent is added to the list below.
func TestWarriorWithOverRankTalentsConstructsWithoutPanicking(t *testing.T) {
	overRanked := []string{
		"improved_bloodrage",    // improvedBloodrageInstantRage
		"improved_execute",      // improvedExecuteRageReduction
		"improved_rend",         // improvedRendDamageMultiplier
		"improved_shield_wall",  // improvedShieldWallDuration
		"defiance",              // defianceThreatMultiplier
		"flurry",                // flurryAttackSpeed, TalentSpellIDs["flurry"]
		"deep_wounds",           // deepWoundsSpellIDs
		"enrage",                // TalentSpellIDs["enrage"]
		"shield_specialization", // TalentSpellIDs["shield_specialization"]
	}

	talentsStr := warrior.ForeverFuryTalents
	for _, name := range overRanked {
		talentsStr = talentStringWithRank(t, talentsStr, name, 9)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("constructing a warrior with every rank-tabled talent at 9 panicked: %v", r)
		}
	}()

	war := buildWarriorForCostTest(t, talentsStr)

	// Improved Execute clamps to its rank-2 value, so Execute costs the
	// client's 15 rage less the talent's 5 rather than reading past the
	// end of the table.
	if got, want := war.Execute.Cost.GetCurrentCost(), 10.0; got != want {
		t.Errorf("Execute costs %v rage with ImprovedExecute=9, want the max-rank %v", got, want)
	}
}

// talentStringWithRank returns talentsStr with one named WarriorTalents
// field set to rank. The field's position is read off the proto rather
// than typed, so the helper cannot fall out of step with the generated
// tree the way a hardcoded offset would.
func talentStringWithRank(t *testing.T, talentsStr string, fieldName string, rank int) string {
	t.Helper()

	fd := (&proto.WarriorTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if fd == nil {
		t.Fatalf("WarriorTalents has no field named %q", fieldName)
	}

	pos := int(fd.Number()) - 1
	treeIdx := 0
	for treeIdx < len(warrior.TalentTreeSizes) && pos >= warrior.TalentTreeSizes[treeIdx] {
		pos -= warrior.TalentTreeSizes[treeIdx]
		treeIdx++
	}
	if treeIdx >= len(warrior.TalentTreeSizes) {
		t.Fatalf("%q is at proto position %d, past the end of the generated tree", fieldName, fd.Number())
	}

	parts := strings.Split(talentsStr, "-")
	chars := []rune(parts[treeIdx])
	chars[pos] = rune('0' + rank)
	parts[treeIdx] = string(chars)
	return strings.Join(parts, "-")
}
