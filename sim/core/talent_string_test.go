package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

func TestTalentsStringFromRanksPlacesNamedFields(t *testing.T) {
	sizes := [3]int{17, 17, 18}
	got, err := TalentsStringFromRanks((&proto.WarriorTalents{}).ProtoReflect(), sizes, map[string]int{
		"improved_heroic_strike": 3,
		"flurry":                 5,
		"shield_slam":            1,
	})
	if err != nil {
		t.Fatal(err)
	}

	back := &proto.WarriorTalents{}
	FillTalentsProto(back.ProtoReflect(), got, sizes)
	if back.ImprovedHeroicStrike != 3 || back.Flurry != 5 || !back.ShieldSlam {
		t.Errorf("round trip through %q lost a rank: %+v", got, back)
	}
	if want := 17 + 1 + 17 + 1 + 18; len(got) != want {
		t.Errorf("string %q is %d characters, want full widths (%d)", got, len(got), want)
	}
}

func TestTalentsStringFromRanksRejectsBadInput(t *testing.T) {
	sizes := [3]int{17, 17, 18}
	msg := (&proto.WarriorTalents{}).ProtoReflect()
	for name, ranks := range map[string]map[string]int{
		"unknown field": {"not_a_talent": 1},
		"rank too big":  {"flurry": 10},
		"negative rank": {"flurry": -1},
	} {
		if _, err := TalentsStringFromRanks(msg, sizes, ranks); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
