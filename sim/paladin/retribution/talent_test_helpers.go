package retribution

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

// talentString builds a full three-segment PaladinTalents talent string
// (Holy-Protection-Retribution, core.FillTalentsProto's positional
// format) granting exactly the {field name: rank} pairs given, and
// zero everywhere else. Field positions are found through reflection on
// PaladinTalents' proto field numbers rather than hard-coded, so a
// client-data regeneration that reorders or resizes a tree cannot make
// this helper silently write to the wrong talent; paladin.TalentTreeSizes
// is the same table core.FillTalentsProto itself slices the string with.
func talentString(t *testing.T, ranks map[string]int) string {
	t.Helper()

	segments := make([][]byte, len(paladin.TalentTreeSizes))
	for i, size := range paladin.TalentTreeSizes {
		segments[i] = []byte(strings.Repeat("0", size))
	}

	offsets := [3]int{0, paladin.TalentTreeSizes[0], paladin.TalentTreeSizes[0] + paladin.TalentTreeSizes[1]}

	for fieldName, rank := range ranks {
		fd := (&proto.PaladinTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(fieldName))
		if fd == nil {
			t.Fatalf("PaladinTalents has no %s field -- talent proto layout changed", fieldName)
		}

		fieldIdx := int(fd.Number()) - 1 // proto field numbers are 1-based
		tree := -1
		for i := len(offsets) - 1; i >= 0; i-- {
			if fieldIdx >= offsets[i] {
				tree = i
				break
			}
		}
		if tree < 0 {
			t.Fatalf("%s field number %d resolved to no tree", fieldName, fd.Number())
		}

		idxInTree := fieldIdx - offsets[tree]
		if idxInTree < 0 || idxInTree >= len(segments[tree]) {
			t.Fatalf("%s resolved to index %d outside its tree (size %d)", fieldName, idxInTree, len(segments[tree]))
		}
		segments[tree][idxInTree] = byte('0' + rank)
	}

	segStrings := make([]string, len(segments))
	for i, seg := range segments {
		segStrings[i] = string(seg)
	}
	return strings.Join(segStrings, "-")
}

// retributionTalentString is talentString for the common case of a
// single Retribution-tree talent; kept for soc_rank_test.go's existing
// call site.
func retributionTalentString(t *testing.T, fieldName string, rank int) string {
	t.Helper()
	return talentString(t, map[string]int{fieldName: rank})
}
