// Package paladintest builds paladin talent strings for tests.
package paladintest

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

// TalentString builds a full three-segment PaladinTalents string (Holy,
// Protection, Retribution) granting exactly the {proto field name: rank}
// pairs given and nothing else. Positions come from the proto's field
// numbers and paladin.TalentTreeSizes, the table core.FillTalentsProto
// slices the string with, so a regenerated tree cannot leave a test
// writing to the wrong talent.
func TalentString(t *testing.T, ranks map[string]int) string {
	t.Helper()

	segments := make([][]byte, len(paladin.TalentTreeSizes))
	offsets := make([]int, len(paladin.TalentTreeSizes))
	for i, size := range paladin.TalentTreeSizes {
		segments[i] = []byte(strings.Repeat("0", size))
		if i > 0 {
			offsets[i] = offsets[i-1] + paladin.TalentTreeSizes[i-1]
		}
	}

	for fieldName, rank := range ranks {
		fd := (&proto.PaladinTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(fieldName))
		if fd == nil {
			t.Fatalf("PaladinTalents has no %s field -- talent proto layout changed", fieldName)
		}
		fieldIdx := int(fd.Number()) - 1 // proto field numbers are 1-based
		tree := len(offsets) - 1
		for tree > 0 && fieldIdx < offsets[tree] {
			tree--
		}
		idxInTree := fieldIdx - offsets[tree]
		if idxInTree >= len(segments[tree]) {
			t.Fatalf("%s resolved to index %d outside its tree (size %d)", fieldName, idxInTree, len(segments[tree]))
		}
		segments[tree][idxInTree] = byte('0' + rank)
	}

	parts := make([]string, len(segments))
	for i, segment := range segments {
		parts[i] = string(segment)
	}
	return strings.Join(parts, "-")
}
