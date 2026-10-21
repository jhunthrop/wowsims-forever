package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// The four fields sim/bulk's Expand reads to decide whether a candidate
// item may go in a slot at all. They are a wire contract with the site's
// data pipeline, which fills them, so the numbers are pinned here.
func TestSimItemCarriesTheExpansionFields(t *testing.T) {
	item := &proto.SimItem{
		Id:                  12640,
		Unique:              true,
		RequiredLevel:       60,
		FactionRestriction:  proto.SimItem_FACTION_RESTRICTION_HORDE_ONLY,
		RandomSuffixOptions: []int32{1825, 1826},
	}

	if !item.GetUnique() {
		t.Error("Unique did not round-trip")
	}
	if got := item.GetRequiredLevel(); got != 60 {
		t.Errorf("RequiredLevel = %d, want 60", got)
	}
	if got := item.GetFactionRestriction(); got != proto.SimItem_FACTION_RESTRICTION_HORDE_ONLY {
		t.Errorf("FactionRestriction = %v, want HORDE_ONLY", got)
	}
	if got := item.GetRandomSuffixOptions(); len(got) != 2 || got[1] != 1826 {
		t.Errorf("RandomSuffixOptions = %v, want [1825 1826]", got)
	}
}

func TestSimItemExpansionFieldNumbers(t *testing.T) {
	fields := (&proto.SimItem{}).ProtoReflect().Descriptor().Fields()
	for name, want := range map[string]int32{
		"unique":                20,
		"required_level":        21,
		"faction_restriction":   22,
		"random_suffix_options": 23,
	} {
		field := fields.ByName(protoreflect.Name(name))
		if field == nil {
			t.Fatalf("SimItem has no field %q", name)
		}
		if got := int32(field.Number()); got != want {
			t.Errorf("SimItem.%s is field %d, want %d", name, got, want)
		}
	}
}

// SimItem.FactionRestriction is redeclared rather than imported from
// UIItem, because common.proto cannot import ui.proto without a cycle.
// Redeclared is only safe while the numbers match, so this pins them
// against UIItem's - if upstream adds a fourth faction value to one and
// not the other, this fails instead of the two silently disagreeing.
func TestSimItemFactionRestrictionMatchesUIItem(t *testing.T) {
	for _, tc := range []struct {
		sim proto.SimItem_FactionRestriction
		ui  proto.UIItem_FactionRestriction
	}{
		{proto.SimItem_FACTION_RESTRICTION_UNSPECIFIED, proto.UIItem_FACTION_RESTRICTION_UNSPECIFIED},
		{proto.SimItem_FACTION_RESTRICTION_ALLIANCE_ONLY, proto.UIItem_FACTION_RESTRICTION_ALLIANCE_ONLY},
		{proto.SimItem_FACTION_RESTRICTION_HORDE_ONLY, proto.UIItem_FACTION_RESTRICTION_HORDE_ONLY},
	} {
		if int32(tc.sim) != int32(tc.ui) {
			t.Errorf("SimItem.%v = %d but UIItem.%v = %d", tc.sim, tc.sim, tc.ui, tc.ui)
		}
	}

	simEnum := (&proto.SimItem{}).ProtoReflect().Descriptor().Enums().ByName("FactionRestriction")
	if simEnum == nil {
		t.Fatal("SimItem has no nested FactionRestriction enum")
	}
	uiEnum := (&proto.UIItem{}).ProtoReflect().Descriptor().Enums().ByName("FactionRestriction")
	if uiEnum == nil {
		t.Fatal("UIItem has no nested FactionRestriction enum")
	}
	if simEnum.Values().Len() != uiEnum.Values().Len() {
		t.Errorf("SimItem.FactionRestriction has %d values, UIItem.FactionRestriction has %d; they must stay in step",
			simEnum.Values().Len(), uiEnum.Values().Len())
	}
}

// ItemFromProto is the module boundary a downstream consumer crosses to
// turn a SimItem its own data lane filled back into a core.Item. A field
// with no home on core.Item is dropped there in silence, which is what
// happened to required_level: this fork has no source for it, but a
// level restriction the site's pipeline does fill must survive the
// crossing. All four expansion fields are checked, not just the three
// this fork can populate itself.
func TestItemFromProtoCarriesEveryExpansionField(t *testing.T) {
	item := ItemFromProto(&proto.SimItem{
		Id:                  12640,
		Unique:              true,
		RequiredLevel:       60,
		FactionRestriction:  proto.SimItem_FACTION_RESTRICTION_HORDE_ONLY,
		RandomSuffixOptions: []int32{1825, 1826},
	})

	if !item.Unique {
		t.Error("Unique did not survive ItemFromProto")
	}
	if item.RequiredLevel != 60 {
		t.Errorf("RequiredLevel = %d, want 60", item.RequiredLevel)
	}
	if item.FactionRestriction != proto.SimItem_FACTION_RESTRICTION_HORDE_ONLY {
		t.Errorf("FactionRestriction = %v, want HORDE_ONLY", item.FactionRestriction)
	}
	if len(item.RandomSuffixOptions) != 2 || item.RandomSuffixOptions[1] != 1826 {
		t.Errorf("RandomSuffixOptions = %v, want [1825 1826]", item.RandomSuffixOptions)
	}
}

// The loaded database carries the three fields the fork already knows.
// This is what stops the copy in database_load.go being written and then
// silently dropped in a later upstream merge.
func TestLoadedDatabaseCarriesTheCopiedFields(t *testing.T) {
	if !WITH_DB {
		t.Skip("item database not loaded; run with --tags=with_db")
	}

	sawUnique, sawSuffixes := false, false
	for _, item := range ItemsByID {
		if item.Unique {
			sawUnique = true
		}
		if len(item.RandomSuffixOptions) > 0 {
			sawSuffixes = true
		}
		if sawUnique && sawSuffixes {
			return
		}
	}
	if !sawUnique {
		t.Error("no loaded item is unique-equipped; the copy in database_load.go is missing")
	}
	if !sawSuffixes {
		t.Error("no loaded item has suffix options; the copy in database_load.go is missing")
	}
}
