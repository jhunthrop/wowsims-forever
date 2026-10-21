package core

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// TalentsStringFromRanks builds the dash-separated talent string
// FillTalentsProto reads, with the named proto fields at the given ranks
// and every other talent at zero. Fields are named as in the generated
// talent proto (snake_case), so a caller never counts positions: a tree
// layout change (a talent added, removed or moved by a patch) cannot
// silently retarget a test or a fixture the way a typed offset does.
//
// Every segment is its tree's full width, which FillTalentsProto accepts
// and which a width check can assert. An unknown field name, a rank that
// does not fit one digit or a field past the last tree is an error.
func TalentsStringFromRanks(data protoreflect.Message, treeSizes [3]int, ranks map[string]int) (string, error) {
	fields := data.Descriptor().Fields()
	segments := [3][]byte{}
	for tree, size := range treeSizes {
		segments[tree] = []byte(strings.Repeat("0", size))
	}

	for name, rank := range ranks {
		field := fields.ByName(protoreflect.Name(name))
		if field == nil {
			return "", fmt.Errorf("talent proto %s has no field %q", data.Descriptor().Name(), name)
		}
		if rank < 0 || rank > 9 {
			return "", fmt.Errorf("talent %q rank %d does not fit one talent-string digit", name, rank)
		}
		tree, index, err := locateTalent(int(field.Number())-1, treeSizes)
		if err != nil {
			return "", fmt.Errorf("talent %q: %w", name, err)
		}
		segments[tree][index] = byte('0' + rank)
	}

	return string(segments[0]) + "-" + string(segments[1]) + "-" + string(segments[2]), nil
}

// locateTalent maps a zero-based proto field position to its tree and
// its index within that tree.
func locateTalent(position int, treeSizes [3]int) (tree, index int, err error) {
	for tree, size := range treeSizes {
		if position < size {
			return tree, position, nil
		}
		position -= size
	}
	return 0, 0, fmt.Errorf("position is past the last tree (sizes %v)", treeSizes)
}
