package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

var curseOfTheElementsRanks = []struct {
	id         int32
	rank       int
	level      int
	manaCost   float64
	damageGain float64 // the multiplier the target takes
	resistance float64 // magic resistance removed
}{
	{440892, 1, 20, 50, 1.04, 30},
	{1311676, 2, 30, 100, 1.06, 45},
	{1311677, 3, 40, 150, 1.08, 60},
	{1311680, 4, 50, 200, 1.10, 75},
}

func TestCurseOfTheElementsRegistersTheClientsFourRanks(t *testing.T) {
	_, built, _ := newBareWarlockForDamageTest(t)

	if len(built.CurseOfElements) != len(curseOfTheElementsRanks) {
		t.Fatalf("level-60 warlock registers %d Curse of the Elements ranks, want %d", len(built.CurseOfElements), len(curseOfTheElementsRanks))
	}
	for i, want := range curseOfTheElementsRanks {
		spell := built.CurseOfElements[i]
		if spell.ActionID.SpellID != want.id || spell.Rank != want.rank || spell.RequiredLevel != want.level {
			t.Errorf("rank slot %d: id %d rank %d level %d, want id %d rank %d level %d",
				i, spell.ActionID.SpellID, spell.Rank, spell.RequiredLevel, want.id, want.rank, want.level)
		}
		if spell.DefaultCast.CastTime != 0 {
			t.Errorf("id %d is not instant", want.id)
		}
	}
}

// curseTargetState is what a Curse of the Elements cast changes on its target.
type curseTargetState struct {
	damageTaken map[stats.SchoolIndex]float64
	resistance  float64
}

func readCurseTargetState(target *core.Unit) curseTargetState {
	state := curseTargetState{damageTaken: map[stats.SchoolIndex]float64{}, resistance: target.GetStat(stats.ShadowResistance)}
	for _, school := range curseOfTheElementsSchools {
		state.damageTaken[school] = target.PseudoStats.SchoolDamageTakenMultiplier[school]
	}
	state.damageTaken[stats.SchoolIndexPhysical] = target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexPhysical]
	return state
}

func castUntilCurseLands(t *testing.T, spellID int32) (*core.Simulation, *Warlock, *core.Unit, curseTargetState) {
	t.Helper()
	for attempt := 0; attempt < landAttempts; attempt++ {
		sim, built, target := newBareWarlockForDamageTest(t)
		before := readCurseTargetState(target)
		spell := spellByID(t, built.CurseOfElements, spellID)
		spell.ApplyEffects(sim, target, spell)
		if built.ActiveCurseAura.Get(target) != nil {
			return sim, built, target, before
		}
	}
	t.Fatalf("curse %d never landed", spellID)
	return nil, nil, nil, curseTargetState{}
}

func TestCurseOfTheElementsRaisesAllMagicDamageTakenByTheRankPercent(t *testing.T) {
	for _, want := range curseOfTheElementsRanks {
		_, _, target, before := castUntilCurseLands(t, want.id)
		after := readCurseTargetState(target)

		for _, school := range curseOfTheElementsSchools {
			got := after.damageTaken[school] / before.damageTaken[school]
			if math.Abs(got-want.damageGain) > 1e-9 {
				t.Errorf("rank %d school %d: damage taken x%.4f, want x%.4f", want.rank, school, got, want.damageGain)
			}
		}
		if after.damageTaken[stats.SchoolIndexPhysical] != before.damageTaken[stats.SchoolIndexPhysical] {
			t.Errorf("rank %d changes physical damage taken", want.rank)
		}
	}
}

func TestCurseOfTheElementsLowersMagicResistanceByTheRankAmount(t *testing.T) {
	for _, want := range curseOfTheElementsRanks {
		_, _, target, before := castUntilCurseLands(t, want.id)
		if got := before.resistance - target.GetStat(stats.ShadowResistance); got != want.resistance {
			t.Errorf("rank %d lowers resistance by %.0f, want %.0f", want.rank, got, want.resistance)
		}
	}
}

// TestCurseOfTheElementsGivesWayToAnotherCurse is the live text: "Only one
// Curse per Warlock can be active on any one target." Casting Curse of Agony
// over it removes the damage-taken bonus and the resistance loss together.
func TestCurseOfTheElementsGivesWayToAnotherCurse(t *testing.T) {
	for attempt := 0; attempt < landAttempts; attempt++ {
		sim, built, target := newBareWarlockForDamageTest(t)
		before := readCurseTargetState(target)

		elements := built.CurseOfElements[len(built.CurseOfElements)-1]
		elements.ApplyEffects(sim, target, elements)
		agony := built.CurseOfAgony[len(built.CurseOfAgony)-1]
		agony.ApplyEffects(sim, target, agony)
		if !agony.Dot(target).IsActive() || built.ActiveCurseAura.Get(target) != agony.Dot(target).Aura {
			continue
		}
		if after := readCurseTargetState(target); after.resistance != before.resistance || after.damageTaken[stats.SchoolIndexShadow] != before.damageTaken[stats.SchoolIndexShadow] {
			t.Fatalf("Curse of Agony landed but the target still carries Curse of the Elements: %+v vs %+v", after, before)
		}
		return
	}
	t.Fatal("Curse of Agony never landed over Curse of the Elements")
}

const landAttempts = 20

// spellByID is the registered spell among spells with the given id.
func spellByID(t *testing.T, spells []*core.Spell, id int32) *core.Spell {
	t.Helper()
	for _, spell := range spells {
		if spell.ActionID.SpellID == id {
			return spell
		}
	}
	t.Fatalf("no registered spell with id %d", id)
	return nil
}
