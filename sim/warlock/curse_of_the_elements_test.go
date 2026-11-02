package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	googleProto "google.golang.org/protobuf/proto"
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
		sim, built, target := newUncursedWarlockForTest(t)
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

// newUncursedWarlockForTest is the damage-test warlock with every raid debuff
// but the raid's own Curse of the Elements, so the warlock's curse is the only
// one on the target.
func newUncursedWarlockForTest(t *testing.T) (*core.Simulation, *Warlock, *core.Unit) {
	t.Helper()
	debuffs := googleProto.Clone(core.FullBuffs.Debuffs).(*proto.Debuffs)
	debuffs.CurseOfElements = false
	return newWarlockWithDebuffsForTest(t, "", debuffs)
}

// landOn applies spell to target until it lands (a curse can miss), failing
// the test if it never does.
func landOn(t *testing.T, sim *core.Simulation, target *core.Unit, spell *core.Spell, landed func() bool) {
	t.Helper()
	for attempt := 0; attempt < landAttempts; attempt++ {
		spell.ApplyEffects(sim, target, spell)
		if landed() {
			return
		}
	}
	t.Fatalf("%v never landed", spell.ActionID)
}

// TestABaneAndACurseHoldSeparateSlots is Forever's split of the old curses:
// every bane's text reads "Only one Bane per Warlock can be active on any one
// target" (Bane of Agony 980 to 11713, Bane of Doom 603, Bane of Havoc
// 1225228) and every curse's "Only one Curse per Warlock can be active on any
// one target" (Curse of the Elements 440892 to 1311680, Recklessness,
// Weakness, Tongues, Exhaustion). So Bane of Agony leaves Curse of the
// Elements on the target, Bane of Doom replaces Agony, and Curse of
// Recklessness replaces the Elements.
func TestABaneAndACurseHoldSeparateSlots(t *testing.T) {
	sim, built, target := newUncursedWarlockForTest(t)
	elements := built.CurseOfElements[len(built.CurseOfElements)-1]
	elementsAura := elements.RelatedAuras[0].Get(target)
	agony := built.CurseOfAgony[len(built.CurseOfAgony)-1]

	landOn(t, sim, target, elements, elementsAura.IsActive)
	landOn(t, sim, target, agony, agony.Dot(target).IsActive)
	if !elementsAura.IsActive() {
		t.Fatal("Bane of Agony removed Curse of the Elements; a bane and a curse share no slot")
	}

	landOn(t, sim, target, built.CurseOfDoom, built.CurseOfDoom.Dot(target).IsActive)
	if agony.Dot(target).IsActive() {
		t.Error("Bane of Doom left Bane of Agony on the target; only one bane per warlock")
	}
	if !elementsAura.IsActive() {
		t.Error("Bane of Doom removed Curse of the Elements")
	}

	recklessness := built.CurseOfRecklessness
	landOn(t, sim, target, recklessness, recklessness.RelatedAuras[0].Get(target).IsActive)
	if elementsAura.IsActive() {
		t.Error("Curse of Recklessness left Curse of the Elements on the target; only one curse per warlock")
	}
	if !built.CurseOfDoom.Dot(target).IsActive() {
		t.Error("Curse of Recklessness removed Bane of Doom")
	}
}

// TestOwnCurseOfTheElementsDoesNotStackWithTheRaidCurse: the raid's Curse of
// the Elements (another warlock's rank 4) and this warlock's are the same
// debuff, so the target never takes both. The warlock does not recast it over
// the raid's, and a forced application leaves the target at one curse's 10%.
func TestOwnCurseOfTheElementsDoesNotStackWithTheRaidCurse(t *testing.T) {
	sim, built, target := newBareWarlockForDamageTest(t)
	before := readCurseTargetState(target)
	elements := built.CurseOfElements[len(built.CurseOfElements)-1]

	if elements.CanCast(sim, target) {
		t.Error("the warlock would cast Curse of the Elements over the raid's")
	}
	elements.RelatedAuras[0].Get(target).Activate(sim)
	after := readCurseTargetState(target)
	for _, school := range curseOfTheElementsSchools {
		if got := after.damageTaken[school] / before.damageTaken[school]; math.Abs(got-1) > 1e-9 {
			t.Errorf("school %d: a second Curse of the Elements multiplies damage taken by %.4f, want 1", school, got)
		}
	}
	if after.resistance != before.resistance {
		t.Errorf("a second Curse of the Elements removes %.0f more resistance, want none", before.resistance-after.resistance)
	}
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
