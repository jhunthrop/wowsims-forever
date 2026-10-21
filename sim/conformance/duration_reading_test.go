package conformance

import (
	"os"
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

func TestOnlyUnmodeledControlEffects(t *testing.T) {
	slow := spellconst.Effect{Effect: effectApplyAura, Aura: auraDecreaseSpeed}
	damage := spellconst.Effect{Effect: effectSchoolDamage}
	periodic := spellconst.Effect{Effect: effectApplyAura, Aura: auraPeriodicDamage}
	interrupt := spellconst.Effect{Effect: effectInterruptCast}

	cases := []struct {
		name    string
		effects []spellconst.Effect
		want    bool
	}{
		{"damage plus slow (Frostbolt)", []spellconst.Effect{slow, damage}, true},
		{"interrupt lockout (Counterspell)", []spellconst.Effect{interrupt}, true},
		{"damage over time is a real duration", []spellconst.Effect{damage, periodic}, false},
		{"slow beside a damage over time", []spellconst.Effect{slow, periodic}, false},
		{"no control effect at all", []spellconst.Effect{damage}, false},
	}
	for _, c := range cases {
		if got := onlyUnmodeledControlEffects(spellconst.Spell{Effects: c.effects}); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func trapClass() spellconst.Class {
	return spellconst.Class{Spells: []spellconst.Spell{
		{ID: 1, Name: "Immolation Trap", Rank: 1, DurationMS: 60000, Effects: []spellconst.Effect{{Effect: 104}}},
		{ID: 2, Name: "Immolation Trap", Rank: 2, DurationMS: 60000, Effects: []spellconst.Effect{{Effect: 104}}},
		{ID: 3, Name: "Immolation Trap", Rank: 1, DurationMS: 15000, Effects: []spellconst.Effect{{Effect: 6, Aura: 3}}},
		{ID: 4, Name: "Immolation Trap", Rank: 2, DurationMS: 16000, Effects: []spellconst.Effect{{Effect: 6, Aura: 3}}},
		{ID: 5, Name: "Freezing Trap", Rank: 1, DurationMS: 60000, Effects: []spellconst.Effect{{Effect: 104}}},
		{ID: 6, Name: "Freezing Trap", Rank: 1, DurationMS: 10000, Effects: []spellconst.Effect{{Effect: 6, Aura: auraStun}}},
	}}
}

// A trap's duration_ms is its armed lifetime; the comparable duration is
// the burn or freeze on the same-rank payload spell (named "<Trap> Effect"
// before build 1.60.1.70291, the trap's own name after).
func TestClientDurationMS_TrapReadsItsEffectSpell(t *testing.T) {
	class := trapClass()
	for _, tc := range []struct {
		id   int32
		want int32
	}{{1, 15000}, {2, 16000}} {
		cast, _ := class.ByID(tc.id)
		if got := durationSpellFor(class, cast).DurationMS; got != tc.want {
			t.Errorf("spell %d: duration %d, want the effect spell's %d", tc.id, got, tc.want)
		}
	}
}

func TestClientDurationMS_PlainSpellKeepsItsOwnDuration(t *testing.T) {
	class := spellconst.Class{Spells: []spellconst.Spell{{ID: 9, Name: "Rip", Rank: 1, DurationMS: 12000, Effects: []spellconst.Effect{{Effect: 6, Aura: 3}}}}}
	rip, _ := class.ByID(9)
	if got := durationSpellFor(class, rip).DurationMS; got != 12000 {
		t.Errorf("Rip duration %d, want 12000", got)
	}
}

// A duration that belongs only to a stun, snare, speed buff or interrupt
// reads as "none" when the preset opts in, and never otherwise.
func TestDurationReadingCoversControlAndMovementForEveryClass(t *testing.T) {
	for _, aura := range []int32{auraDecreaseSpeed, auraIncreaseSpeed, auraStun} {
		client := spellconst.Spell{DurationMS: 5000, Effects: []spellconst.Effect{{Effect: effectApplyAura, Aura: aura}}}
		for _, slug := range []string{"hunter", "rogue", "warrior", "mage"} {
			if got := durationReading(Row{ClassSlug: slug}, client, nil, nil); got != reasonUnmodeledEffect {
				t.Errorf("%s aura %d reading = %q, want the unmodeled-effect reason", slug, aura, got)
			}
		}
	}
}

func TestDurationReadingKeepsPerSpellReason(t *testing.T) {
	client := spellconst.Spell{Name: "Earth Shock", DurationMS: 2000, Effects: []spellconst.Effect{{Effect: effectInterruptCast}}}
	got := durationReading(Row{ClassSlug: "shaman"}, client, nil, nil)
	if got != unsimulatedDurations["shaman/Earth Shock"] || got == "" {
		t.Errorf("Earth Shock reading = %q, want its own lock-out reason", got)
	}
}

func TestDurationReadingKeepsRealDurations(t *testing.T) {
	client := spellconst.Spell{DurationMS: 12000, Effects: []spellconst.Effect{{Effect: effectApplyAura, Aura: auraPeriodicDamage}}}
	if got := durationReading(Row{ClassSlug: "druid"}, client, nil, nil); got != "" {
		t.Errorf("a damage over time reads %q, want it compared as numbers", got)
	}
}

func TestClassifiedTrainablesAreAllDispositioned(t *testing.T) {
	for slug := range trainableDispositions {
		golden, err := os.ReadFile(goldenPath(slug))
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		if strings.Contains(string(golden), "| "+unclassifiedDisposition+" |") {
			t.Errorf("%s golden has an ability with no disposition in dispositions.go", slug)
		}
	}
}

func TestCompareTrainables_ModeledBuffIsNotMissing(t *testing.T) {
	trainables := ClassTrainables{
		ClassSlug: "druid",
		Trainables: []Trainable{
			{Name: "Mark of the Wild", Source: "skill_line_ability", Active: true, Cost: 1, Ranks: []TrainableRank{{ID: 1126, Rank: 1, Level: 1}}},
			{Name: "Gift of the Wild", Source: "skill_line_ability", Active: true, Cost: 1, Ranks: []TrainableRank{{ID: 21849, Rank: 1, Level: 50}}},
			{Name: "Growl", Source: "skill_line_ability", Active: true, CooldownMS: 8000, Ranks: []TrainableRank{{ID: 6795, Rank: 1, Level: 10}}},
			{Name: "Rake", Source: "skill_line_ability", Active: true, Cost: 40, Ranks: []TrainableRank{{ID: 1822, Rank: 1, Level: 24}}},
		},
	}
	gaps := compareTrainables(trainables, map[int32]bool{1822: true})
	if gaps.Active != 4 {
		t.Errorf("active = %d, want 4", gaps.Active)
	}
	if len(gaps.ModeledAsBuff) != 2 {
		t.Errorf("modelled as buff = %d, want Mark and Gift of the Wild", len(gaps.ModeledAsBuff))
	}
	if len(gaps.Unregistered) != 1 || gaps.Unregistered[0].Name != "Growl" {
		t.Errorf("unregistered = %+v, want only Growl", gaps.Unregistered)
	}
}

func TestRacialNoteAppendsToWhy(t *testing.T) {
	starshards := Trainable{Name: "Starshards", Cost: 1, CooldownMS: 30000}
	if got := starshards.whyAndRacial("priest"); got == starshards.whyItMatters() {
		t.Errorf("Starshards reads %q, want its racial note appended", got)
	}
	if got := starshards.whyAndRacial("druid"); got != starshards.whyItMatters() {
		t.Errorf("a druid's Starshards reads %q, want no note", got)
	}
}
