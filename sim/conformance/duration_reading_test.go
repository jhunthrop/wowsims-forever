package conformance

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/spellconst"
)

func trapClass() spellconst.Class {
	return spellconst.Class{Spells: []spellconst.Spell{
		{ID: 1, Name: "Immolation Trap", Rank: 1, DurationMS: 60000, Effects: []spellconst.Effect{{Effect: 104}}},
		{ID: 2, Name: "Immolation Trap", Rank: 2, DurationMS: 60000, Effects: []spellconst.Effect{{Effect: 104}}},
		{ID: 3, Name: "Immolation Trap Effect", Rank: 1, DurationMS: 15000, Effects: []spellconst.Effect{{Effect: 6, Aura: 3}}},
		{ID: 4, Name: "Immolation Trap Effect", Rank: 2, DurationMS: 16000, Effects: []spellconst.Effect{{Effect: 6, Aura: 3}}},
		{ID: 5, Name: "Freezing Trap", Rank: 1, DurationMS: 60000, Effects: []spellconst.Effect{{Effect: 104}}},
		{ID: 6, Name: "Freezing Trap Effect", Rank: 1, DurationMS: 10000, Effects: []spellconst.Effect{{Effect: 6, Aura: auraStun}}},
	}}
}

// A trap's duration_ms is its armed lifetime; the comparable duration is
// the burn or freeze on the same-rank "<Trap> Effect" spell.
func TestClientDurationMS_TrapReadsItsEffectSpell(t *testing.T) {
	class := trapClass()
	for _, tc := range []struct {
		id   int32
		want int32
	}{{1, 15000}, {2, 16000}} {
		cast, _ := class.ByID(tc.id)
		if got := clientDurationMS(class, cast, false); got != tc.want {
			t.Errorf("spell %d: duration %d, want the effect spell's %d", tc.id, got, tc.want)
		}
	}
}

func TestClientDurationMS_PlainSpellKeepsItsOwnDuration(t *testing.T) {
	class := spellconst.Class{Spells: []spellconst.Spell{{ID: 9, Name: "Rip", Rank: 1, DurationMS: 12000, Effects: []spellconst.Effect{{Effect: 6, Aura: 3}}}}}
	rip, _ := class.ByID(9)
	if got := clientDurationMS(class, rip, true); got != 12000 {
		t.Errorf("Rip duration %d, want 12000", got)
	}
}

// A duration that belongs only to a stun, snare, speed buff or interrupt
// reads as "none" when the preset opts in, and never otherwise.
func TestClientDurationMS_MovementAndControlOnlyDurationsAreSkippedOnOptIn(t *testing.T) {
	class := trapClass()
	freezing, _ := class.ByID(5)
	if got := clientDurationMS(class, freezing, true); got != 0 {
		t.Errorf("opted in: Freezing Trap duration %d, want 0", got)
	}
	if got := clientDurationMS(class, freezing, false); got != 10000 {
		t.Errorf("not opted in: Freezing Trap duration %d, want 10000", got)
	}

	for name, spell := range map[string]spellconst.Spell{
		"snare plus damage": {DurationMS: 10000, Effects: []spellconst.Effect{{Effect: 6, Aura: auraDecreaseSpeed}, {Effect: 2}}},
		"speed buff":        {DurationMS: 3000, Effects: []spellconst.Effect{{Effect: 31}, {Effect: 6, Aura: auraIncreaseSpeed}}},
		"interrupt":         {DurationMS: 5000, Effects: []spellconst.Effect{{Effect: 2}, {Effect: effectInterrupt}}},
	} {
		if !durationIsUnsimmedMovement(spell) {
			t.Errorf("%s: duration not treated as movement or control", name)
		}
	}
	for name, spell := range map[string]spellconst.Spell{
		"damage over time":       {DurationMS: 12000, Effects: []spellconst.Effect{{Effect: 6, Aura: 3}}},
		"snare plus a real aura": {DurationMS: 8000, Effects: []spellconst.Effect{{Effect: 6, Aura: auraDecreaseSpeed}, {Effect: 6, Aura: 3}}},
		"no aura at all":         {DurationMS: 8000, Effects: []spellconst.Effect{{Effect: 2}}},
	} {
		if durationIsUnsimmedMovement(spell) {
			t.Errorf("%s: duration wrongly treated as movement or control", name)
		}
	}
}

func TestDurationTarget(t *testing.T) {
	if got := durationTarget(Preset{}, nil); got != nil {
		t.Errorf("no caster: target %v, want nil", got)
	}
	ally := &core.Unit{Type: core.PlayerUnit}
	caster := &core.Unit{CurrentTarget: ally}
	if got := durationTarget(Preset{}, caster); got != ally {
		t.Errorf("default preset reads %v, want the caster's current target", got)
	}
	// Opted in, a caster without an environment has no enemy list to
	// search and keeps its current target.
	if got := durationTarget(Preset{ReadDebuffsOnEnemy: true}, caster); got != ally {
		t.Errorf("ReadDebuffsOnEnemy without an environment: target %v, want the current target", got)
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
