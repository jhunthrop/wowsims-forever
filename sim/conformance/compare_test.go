package conformance

import (
	"strings"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// newFakeSpell builds a minimal *core.Spell/*core.Unit pair for
// engineDuration's tests: no Simulation, no Environment, no
// RegisterSpell - just the exported fields engineDuration and
// matchingAura actually read. RegisterAura is safe to call on a
// zero-value *core.Unit because its only guard
// ("unit.Env != nil && unit.Env.IsFinalized()") is false when Env is
// nil, exactly as it is here.
func newFakeSpell(spellID int32) (*core.Spell, *core.Unit) {
	caster := &core.Unit{}
	spell := &core.Spell{
		ActionID: core.ActionID{SpellID: spellID},
		Unit:     caster,
	}
	return spell, caster
}

func TestEngineDuration_RelatedSelfBuffWins(t *testing.T) {
	spell, caster := newFakeSpell(100)
	spell.RelatedSelfBuff = caster.RegisterAura(core.Aura{
		Label:    "self buff",
		Duration: 7 * time.Second,
	})

	// A caster aura that would also match, so the test fails loudly if
	// RelatedSelfBuff ever stops taking priority over the wider search.
	caster.RegisterAura(core.Aura{
		Label:    "decoy",
		ActionID: core.ActionID{SpellID: 100},
		Duration: 99 * time.Second,
	})

	ms, found := engineDuration(spell, nil, map[int32]bool{100: true})
	if !found || ms != 7000 {
		t.Fatalf("engineDuration = (%d, %v), want (7000, true)", ms, found)
	}
}

func TestEngineDuration_FindsCasterAuraBySiblingRank(t *testing.T) {
	// The spell under test is rank 2 (SpellID 201), but the aura its
	// cast actually activates is pinned to rank 5's SpellID (205) -
	// exactly the shape core.StrengthOfEarthTotemAura uses (always the
	// top rank's ActionID, regardless of which rank cast it). Only
	// passing every sibling rank's SpellID, not just 201, finds it.
	spell, caster := newFakeSpell(201)
	caster.RegisterAura(core.Aura{
		Label:    "Strength of Earth Totem",
		ActionID: core.ActionID{SpellID: 205},
		Duration: 120 * time.Second,
	})

	siblingIDs := map[int32]bool{201: true, 205: true}

	if ms, found := engineDuration(spell, nil, siblingIDs); !found || ms != 120000 {
		t.Fatalf("engineDuration = (%d, %v), want (120000, true)", ms, found)
	}

	// Passing only the spell's own id (the pre-Job-1 behavior's
	// equivalent) must NOT find it - this is the exact gap Job 1 closes.
	if ms, found := engineDuration(spell, nil, map[int32]bool{201: true}); found || ms != 0 {
		t.Fatalf("engineDuration with no sibling ids = (%d, %v), want (0, false)", ms, found)
	}
}

func TestEngineDuration_FindsTargetAura(t *testing.T) {
	spell, caster := newFakeSpell(300)
	target := &core.Unit{}
	caster.CurrentTarget = target

	target.RegisterAura(core.Aura{
		Label:    "Expose Armor",
		ActionID: core.ActionID{SpellID: 300},
		Duration: 30 * time.Second,
	})

	ms, found := engineDuration(spell, nil, map[int32]bool{300: true})
	if !found || ms != 30000 {
		t.Fatalf("engineDuration = (%d, %v), want (30000, true)", ms, found)
	}
}

func TestEngineDuration_FindsPetAura(t *testing.T) {
	spell, _ := newFakeSpell(400)
	character := &core.Character{}
	pet := &core.Pet{}
	character.Pets = []*core.Pet{pet}

	pet.RegisterAura(core.Aura{
		Label:    "pet buff",
		ActionID: core.ActionID{SpellID: 400},
		Duration: 18 * time.Second,
	})

	ms, found := engineDuration(spell, character, map[int32]bool{400: true})
	if !found || ms != 18000 {
		t.Fatalf("engineDuration = (%d, %v), want (18000, true)", ms, found)
	}
}

func TestEngineDuration_NoAuraAnywhereReportsNotFound(t *testing.T) {
	spell, caster := newFakeSpell(500)
	target := &core.Unit{}
	caster.CurrentTarget = target
	character := &core.Character{}
	pet := &core.Pet{}
	character.Pets = []*core.Pet{pet}

	// An aura exists, but for a completely different spell - it must
	// not match.
	caster.RegisterAura(core.Aura{
		Label:    "unrelated",
		ActionID: core.ActionID{SpellID: 999},
		Duration: time.Second,
	})

	ms, found := engineDuration(spell, character, map[int32]bool{500: true})
	if found || ms != 0 {
		t.Fatalf("engineDuration = (%d, %v), want (0, false)", ms, found)
	}
}

// TestEngineDuration_NeverExpiresReportsClientSentinel guards the
// regression Job 1's wider search almost introduced: a proc-style aura
// with no natural timeout (Inner Focus, Shadowform) carries
// core.NeverExpires (math.MaxInt64 nanoseconds), which must be read as
// the client's own -1 ("until removed") sentinel, not truncated into an
// int32 millisecond count.
func TestEngineDuration_NeverExpiresReportsClientSentinel(t *testing.T) {
	spell, caster := newFakeSpell(600)
	spell.RelatedSelfBuff = caster.RegisterAura(core.Aura{
		Label:    "permanent buff",
		Duration: core.NeverExpires,
	})

	ms, found := engineDuration(spell, nil, map[int32]bool{600: true})
	if !found || ms != -1 {
		t.Fatalf("engineDuration = (%d, %v), want (-1, true)", ms, found)
	}
}

func TestSiblingSpellIDs(t *testing.T) {
	clientClass := spellconst.Class{
		Spells: []spellconst.Spell{
			{ID: 1, Name: "Strength of Earth Totem", Rank: 1},
			{ID: 2, Name: "Strength of Earth Totem", Rank: 2},
			{ID: 3, Name: "Something Else", Rank: 1},
		},
	}

	ids := siblingSpellIDs(clientClass, "Strength of Earth Totem")
	if len(ids) != 2 || !ids[1] || !ids[2] {
		t.Fatalf("siblingSpellIDs = %v, want {1:true, 2:true}", ids)
	}
	if ids[3] {
		t.Fatalf("siblingSpellIDs leaked an unrelated spell's id: %v", ids)
	}
}

// TestRowFor_DropsGenericAttackRow is Job 3: the client's "Attack"
// entry is the generic melee-swing button, not a real player spell, and
// is excluded from the comparison entirely - even when, as with
// shaman's Searing Totem, an engine spell happens to share its SpellID.
func TestRowFor_DropsGenericAttackRow(t *testing.T) {
	clientClass := spellconst.Class{
		Spells: []spellconst.Spell{
			{ID: 3606, Name: "Attack", Rank: 1, CastTimeMS: 2200, SpellLevel: 10},
		},
	}
	spell, _ := newFakeSpell(3606)

	_, ok := rowFor(clientClass, Preset{Label: "test"}, 60, nil, spell)
	if ok {
		t.Fatalf("rowFor returned a row for the client's generic Attack entry; Job 3 requires it be dropped")
	}
}

// TestNoAuraNote checks the Job 1 "new column or note" requirement: a
// duration mismatch where engineDuration found nothing at all is
// annotated distinctly from one where an aura was found but the two
// sides' numbers simply disagree.
func TestNoAuraNote(t *testing.T) {
	notFound := Row{
		ClientRequiredLevel: 0, EngineRequiredLevel: 0,
		HasDuration: true, ClientDurationMS: 60000, EngineDurationMS: 0, EngineDurationFound: false,
	}
	_, diff := verdictFor(notFound)
	if !containsAll(diff, "duration_ms 60000->0", "no aura registered") {
		t.Fatalf("Diff = %q, want it to name the missing aura", diff)
	}

	found := Row{
		HasDuration: true, ClientDurationMS: 60000, EngineDurationMS: 30000, EngineDurationFound: true,
	}
	_, diff = verdictFor(found)
	if !containsAll(diff, "duration_ms 60000->30000") {
		t.Fatalf("Diff = %q, want the plain mismatch", diff)
	}
	if containsAll(diff, "no aura registered") {
		t.Fatalf("Diff = %q, should not claim no aura was registered when one was found", diff)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestVerdictFor_UnsimulatedDurationIsNamedNotMismatched(t *testing.T) {
	row := Row{
		ClientDurationMS:     2000,
		HasDuration:          true,
		DurationNotSimulated: unsimulatedDurationReason("shaman", "Earth Shock"),
	}
	if row.DurationNotSimulated == "" {
		t.Fatal("Earth Shock's lock-out duration should be a named unsimulated effect")
	}
	verdict, diff := verdictFor(row)
	if verdict != VerdictDurationNotSimulated || !containsAll(diff, "2000", "lock-out") {
		t.Fatalf("verdictFor = (%q, %q), want the %q verdict naming the lock-out", verdict, diff, VerdictDurationNotSimulated)
	}

	row.ClientGCDMS = 1500
	if verdict, _ := verdictFor(row); verdict != "mismatch" {
		t.Fatalf("a row whose GCD also differs must stay a mismatch, got %q", verdict)
	}
}

// TestShamanGoldenHasNoMismatchRows holds the shaman at zero real
// mismatches in both tables: a row that differs from the client must be
// fixed in the engine or named as an unsimulated effect in
// unsimulated.go, never left unexplained.
func TestShamanGoldenHasNoMismatchRows(t *testing.T) {
	golden, err := readGolden("shaman")
	if err != nil {
		t.Fatalf("reading the shaman golden: %v", err)
	}
	for _, line := range strings.Split(golden, "\n") {
		if strings.Contains(line, "| mismatch |") {
			t.Errorf("shaman golden carries a mismatch row: %s", line)
		}
	}
}
