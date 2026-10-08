package holy

import (
	"slices"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// healingSpells are the healing spells a Holy Paladin learns, by the ids of
// their ranks in learning order, with the talent each needs (none for a
// trainable that is not gated).
var healingSpells = []struct {
	name   string
	ids    []int32
	talent string
}{
	{"Holy Light", holyLightRankIDs, ""},
	{"Flash of Light", flashOfLightRankIDs, ""},
	{"Blessing of Light", []int32{19977, 19978, 19979}, ""},
	{"Greater Blessing of Light", []int32{25890}, ""},
	{"Holy Shock", holyShockCastIDs, "holy_shock"},
	{"Holy Shock (heal)", holyShockHealIDs, "holy_shock"},
	{"Light's Vigil", lightsVigilCastIDs, "lights_vigil"},
	{"Light's Vigil (heal)", lightsVigilHealIDs, "lights_vigil"},
}

var allHealingTalents = map[string]int{"holy_shock": 1, "lights_vigil": 1, "divine_favor": 1}

// learnedAt is the ranks the client lets a character of the given level
// learn.
func learnedAt(t *testing.T, ids []int32, level int32) []int32 {
	t.Helper()
	class, err := spellconst.Load(clientPaladinSpellconst)
	if err != nil {
		t.Fatalf("loading the client table: %v", err)
	}
	var learned []int32
	for _, id := range ids {
		spell, ok := class.ByID(id)
		if !ok {
			t.Fatalf("spell %d is not in the client table", id)
		}
		if int32(spell.SpellLevel) <= level {
			learned = append(learned, id)
		}
	}
	return learned
}

func registeredAt(healer *core.Character, ids []int32) []int32 {
	var registered []int32
	for _, id := range ids {
		if healer.GetSpell(core.ActionID{SpellID: id}) != nil {
			registered = append(registered, id)
		}
	}
	return registered
}

func TestARankRegistersOnlyWhenTheCharacterHasLearnedIt(t *testing.T) {
	for _, level := range core.LevelSmokeLevels {
		healer, _ := unitsOf(t, fight{level: level, talents: allHealingTalents})
		for _, spells := range healingSpells {
			want := learnedAt(t, spells.ids, level)
			if got := registeredAt(healer, spells.ids); !slices.Equal(got, want) {
				t.Errorf("level %d %s: registered %v, want %v", level, spells.name, got, want)
			}
		}
	}
}

func TestGatedHealsNeedTheirTalent(t *testing.T) {
	healer, _ := unitsOf(t, fight{})
	for _, spells := range healingSpells {
		if spells.talent == "" {
			continue
		}
		if got := registeredAt(healer, spells.ids); len(got) != 0 {
			t.Errorf("%s registered without %s: %v", spells.name, spells.talent, got)
		}
	}
}

func TestLevelSmoke(t *testing.T) {
	core.RunLevelSmoke(t, core.LevelSmokePreset{
		Label:   "HolyPaladin",
		Class:   proto.Class_ClassPaladin,
		Race:    proto.Race_RaceHuman,
		Talents: paladinTalents(t, allHealingTalents),
		SpecOptions: &proto.Player_HolyPaladin{HolyPaladin: &proto.HolyPaladin{
			Options: &proto.PaladinOptions{PrimarySeal: proto.PaladinSeal_Righteousness},
		}},
	})
}
